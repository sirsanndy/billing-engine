package postgres

import (
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/infrastructure/monitoring"
	"billing-engine/internal/pkg/apperrors"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoanRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewLoanRepository(db *pgxpool.Pool, logger *slog.Logger) *LoanRepository {
	return &LoanRepository{db: db, logger: logger.With("component", "LoanRepository")}
}

func (r *LoanRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	return tx, nil
}

func (r *LoanRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {
	err := tx.Commit(ctx)
	if err != nil {

		r.logger.ErrorContext(ctx, "Failed to commit transaction", "error", err)
		return fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	return nil
}

func (r *LoanRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	err := tx.Rollback(ctx)

	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		r.logger.ErrorContext(ctx, "Failed to rollback transaction", "error", err)

		return fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	return nil
}

func (r *LoanRepository) CreateLoan(ctx context.Context, newLoan *loan.Loan, schedule []loan.ScheduleEntry) (*loan.Loan, error) {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer r.RollbackTx(ctx, tx)

	loanSQL := `
        INSERT INTO loans (principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        RETURNING id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at`

	var createdLoan loan.Loan
	err = tx.QueryRow(ctx, loanSQL,
		newLoan.PrincipalAmount, newLoan.InterestRate, newLoan.TermWeeks, newLoan.WeeklyPaymentAmount,
		newLoan.TotalLoanAmount, newLoan.StartDate, newLoan.Status,
	).Scan(
		&createdLoan.ID, &createdLoan.PrincipalAmount, &createdLoan.InterestRate, &createdLoan.TermWeeks,
		&createdLoan.WeeklyPaymentAmount, &createdLoan.TotalLoanAmount, &createdLoan.StartDate,
		&createdLoan.Status, &createdLoan.CreatedAt, &createdLoan.UpdatedAt,
	)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to insert loan", "error", err)

		return nil, fmt.Errorf("%w: failed to insert loan: %w", apperrors.ErrDatabase, err)
	}
	r.logger.InfoContext(ctx, "Loan created in DB", "loan_id", createdLoan.ID)

	if len(schedule) > 0 {
		scheduleSQL := `
            INSERT INTO loan_schedule (loan_id, week_number, due_date, due_amount, status, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`

		batch := &pgx.Batch{}
		for _, entry := range schedule {
			batch.Queue(scheduleSQL, createdLoan.ID, entry.WeekNumber, entry.DueDate, entry.DueAmount, entry.Status)
		}

		results := tx.SendBatch(ctx, batch)

		for i := 0; i < len(schedule); i++ {
			_, err = results.Exec()
			if err != nil {
				results.Close()
				r.logger.ErrorContext(ctx, "Failed executing schedule batch insert", "error", err, "entry_index", i, "loan_id", createdLoan.ID)
				return nil, fmt.Errorf("%w: failed inserting schedule entry %d: %w", apperrors.ErrDatabase, i+1, err)
			}
		}
		err = results.Close()
		if err != nil {
			r.logger.ErrorContext(ctx, "Failed closing schedule batch results", "error", err, "loan_id", createdLoan.ID)
			return nil, fmt.Errorf("%w: closing batch results failed: %w", apperrors.ErrDatabase, err)
		}
	}
	r.logger.InfoContext(ctx, "Loan schedule created in DB", "loan_id", createdLoan.ID, "num_entries", len(schedule))

	if err := r.CommitTx(ctx, tx); err != nil {
		return nil, err
	}

	return &createdLoan, nil
}

func (r *LoanRepository) GetLoanByID(ctx context.Context, loanID int64) (*loan.Loan, error) {
	query := `
        SELECT id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at
        FROM loans
        WHERE id = $1`
	status := "success"
	startTime := time.Now()

	var l loan.Loan
	err := r.db.QueryRow(ctx, query, loanID).Scan(
		&l.ID, &l.PrincipalAmount, &l.InterestRate, &l.TermWeeks,
		&l.WeeklyPaymentAmount, &l.TotalLoanAmount, &l.StartDate,
		&l.Status, &l.CreatedAt, &l.UpdatedAt,
	)

	if err != nil {
		status = "error"
	}
	monitoring.RecordDBQuery("GetLoanByID", status, time.Since(startTime))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WarnContext(ctx, "Loan not found", "loan_id", loanID)
			return nil, apperrors.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "Failed to get loan by ID", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	return &l, nil
}

func (r *LoanRepository) GetScheduleByLoanID(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1
        ORDER BY week_number ASC`

	rows, err := r.db.Query(ctx, query, loanID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to query loan schedule", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	defer rows.Close()

	schedule := make([]loan.ScheduleEntry, 0)
	for rows.Next() {
		var entry loan.ScheduleEntry
		err := rows.Scan(
			&entry.ID, &entry.LoanID, &entry.WeekNumber, &entry.DueDate,
			&entry.DueAmount, &entry.PaidAmount, &entry.PaymentDate,
			&entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			r.logger.ErrorContext(ctx, "Failed to scan schedule row", "loan_id", loanID, "error", err)
			return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
		}
		schedule = append(schedule, entry)
	}

	if err = rows.Err(); err != nil {
		r.logger.ErrorContext(ctx, "Error iterating schedule rows", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}

	return schedule, nil
}

func (r *LoanRepository) GetUnpaidSchedules(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID' -- PENDING or MISSED
        ORDER BY due_date ASC`

	rows, err := r.db.Query(ctx, query, loanID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to query unpaid schedules", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	defer rows.Close()

	schedule := make([]loan.ScheduleEntry, 0)
	for rows.Next() {
		var entry loan.ScheduleEntry
		err := rows.Scan(
			&entry.ID, &entry.LoanID, &entry.WeekNumber, &entry.DueDate,
			&entry.DueAmount, &entry.PaidAmount, &entry.PaymentDate,
			&entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			r.logger.ErrorContext(ctx, "Failed to scan unpaid schedule row", "loan_id", loanID, "error", err)
			return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
		}
		schedule = append(schedule, entry)
	}

	if err = rows.Err(); err != nil {
		r.logger.ErrorContext(ctx, "Error iterating unpaid schedule rows", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}

	return schedule, nil
}

func (r *LoanRepository) GetLastTwoDueUnpaidSchedules(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID' -- PENDING or MISSED
        ORDER BY due_date DESC -- Order by due date DESC
        LIMIT 2`

	rows, err := r.db.Query(ctx, query, loanID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to query last two unpaid schedules", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	defer rows.Close()

	schedule := make([]loan.ScheduleEntry, 0, 2)
	for rows.Next() {
		var entry loan.ScheduleEntry
		err := rows.Scan(
			&entry.ID, &entry.LoanID, &entry.WeekNumber, &entry.DueDate,
			&entry.DueAmount, &entry.PaidAmount, &entry.PaymentDate,
			&entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			r.logger.ErrorContext(ctx, "Failed to scan last two unpaid schedule row", "loan_id", loanID, "error", err)
			return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
		}
		schedule = append(schedule, entry)
	}

	if err = rows.Err(); err != nil {
		r.logger.ErrorContext(ctx, "Error iterating last two unpaid schedule rows", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}

	return schedule, nil
}

func (r *LoanRepository) FindOldestUnpaidEntryForUpdate(ctx context.Context, tx pgx.Tx, loanID int64) (*loan.ScheduleEntry, error) {

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status = 'PENDING'
        ORDER BY due_date ASC
        LIMIT 1
        FOR UPDATE`

	var entry loan.ScheduleEntry
	err := tx.QueryRow(ctx, query, loanID).Scan(
		&entry.ID, &entry.LoanID, &entry.WeekNumber, &entry.DueDate,
		&entry.DueAmount, &entry.PaidAmount, &entry.PaymentDate,
		&entry.Status, &entry.CreatedAt, &entry.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {

			r.logger.InfoContext(ctx, "No pending schedule entry found for update", "loan_id", loanID)
			return nil, apperrors.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "Failed to find/lock oldest unpaid schedule entry", "loan_id", loanID, "error", err)
		return nil, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	return &entry, nil
}

func (r *LoanRepository) UpdateScheduleEntryInTx(ctx context.Context, tx pgx.Tx, entry *loan.ScheduleEntry) error {
	sql := `
        UPDATE loan_schedule
        SET paid_amount = $1, payment_date = $2, status = $3, updated_at = NOW()
        WHERE id = $4 AND loan_id = $5`

	cmdTag, err := tx.Exec(ctx, sql, entry.PaidAmount, entry.PaymentDate, entry.Status, entry.ID, entry.LoanID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to update schedule entry", "entry_id", entry.ID, "loan_id", entry.LoanID, "error", err)
		return fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	if cmdTag.RowsAffected() != 1 {
		r.logger.ErrorContext(ctx, "Schedule entry update affected zero rows", "entry_id", entry.ID, "loan_id", entry.LoanID)

		return fmt.Errorf("%w: schedule entry update affected zero rows", apperrors.ErrDatabase)
	}
	return nil
}

func (r *LoanRepository) UpdateLoanStatusInTx(ctx context.Context, tx pgx.Tx, loanID int64, status loan.LoanStatus) error {
	sql := `UPDATE loans SET status = $1, updated_at = NOW() WHERE id = $2`
	cmdTag, err := tx.Exec(ctx, sql, status, loanID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to update loan status", "loan_id", loanID, "status", status, "error", err)
		return fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	if cmdTag.RowsAffected() != 1 {
		r.logger.ErrorContext(ctx, "Loan status update affected zero rows", "loan_id", loanID, "status", status)
		return fmt.Errorf("%w: loan status update affected zero rows", apperrors.ErrDatabase)
	}
	r.logger.InfoContext(ctx, "Loan status updated in DB", "loan_id", loanID, "new_status", status)
	return nil
}

func (r *LoanRepository) CheckIfAllPaymentsMadeInTx(ctx context.Context, tx pgx.Tx, loanID int64) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM loan_schedule WHERE loan_id = $1 AND status != 'PAID'`
	err := tx.QueryRow(ctx, query, loanID).Scan(&count)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to count non-paid schedule entries", "loan_id", loanID, "error", err)
		return false, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
	}
	return count == 0, nil
}

func (r *LoanRepository) GetTotalOutstandingAmount(ctx context.Context, loanID int64) (float64, error) {
	var totalOutstanding float64

	query := `
        SELECT COALESCE(SUM(due_amount - paid_amount), 0.00)
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID'`

	err := r.db.QueryRow(ctx, query, loanID).Scan(&totalOutstanding)
	if err != nil {

		if !errors.Is(err, pgx.ErrNoRows) {
			r.logger.ErrorContext(ctx, "Failed to calculate total outstanding amount", "loan_id", loanID, "error", err)
			return 0, fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
		}
	}

	if totalOutstanding < 0 {
		r.logger.WarnContext(ctx, "Calculated outstanding amount is negative, returning 0", "loan_id", loanID, "calculated_value", totalOutstanding)
		return 0, nil
	}

	return totalOutstanding, nil
}

func translateDBError(err error, contextLogger *slog.Logger) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {

		if pgErr.Code == "23505" {
			contextLogger.Warn("Database unique constraint violation", "detail", pgErr.Detail, "constraint", pgErr.ConstraintName)
			return fmt.Errorf("%w: %s", apperrors.ErrAlreadyExists, pgErr.ConstraintName)
		}

		contextLogger.Error("PostgreSQL specific error", "code", pgErr.Code, "message", pgErr.Message, "detail", pgErr.Detail)
		return fmt.Errorf("%w: db error code %s", apperrors.ErrDatabase, pgErr.Code)
	}

	contextLogger.Error("Generic database error", "error", err)
	return fmt.Errorf("%w: %w", apperrors.ErrDatabase, err)
}
