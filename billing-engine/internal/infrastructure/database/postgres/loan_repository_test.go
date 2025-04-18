package postgres

import (
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/pkg/apperrors"
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	args := m.Called(ctx)
	return args.Get(0).(*pgxpool.Conn), args.Error(1)
}

func (m *MockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return m.Called(ctx, sql, args).Get(0).(pgx.Row)
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	argsMock := m.Called(ctx, sql, args)
	return argsMock.Get(0).(pgx.Rows), argsMock.Error(1)
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	argsMock := m.Called(ctx, sql, args)
	return argsMock.Get(0).(pgconn.CommandTag), argsMock.Error(1)
}

func (m *MockDB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

var logger = slog.New(slog.NewTextHandler(io.Discard, nil))

const pgxmockExpectationsNotMetMsg = "pgxmock expectations not met"

func setupLoanRepo(t *testing.T) (context.Context, *LoanRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err, "Failed to create mock pool")
	repo := NewLoanRepository(mockPool, logger)
	ctx := context.Background()

	return ctx, repo, mockPool
}

func TestLoanRepositoryBeginTxSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	mockPool.ExpectBegin()

	tx, err := repo.BeginTx(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryBeginTxError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	dbErr := errors.New("db begin failed")
	mockPool.ExpectBegin().WillReturnError(dbErr)

	tx, err := repo.BeginTx(ctx)

	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryCommitTxSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectCommit()

	err = repo.CommitTx(ctx, mockTx)

	assert.NoError(t, err)

}

func TestLoanRepositoryCommitTxError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectCommit().WillReturnError(pgx.ErrTxClosed)
	err = repo.CommitTx(ctx, mockTx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
}

func TestLoanRepositoryRollbackTxSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectRollback()

	err = repo.RollbackTx(ctx, mockTx)

	assert.NoError(t, err)
}

func TestLoanRepositoryRollbackTxError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	dbErr := errors.New("db rollback failed")
	mockTx.(pgxmock.PgxCommonIface).ExpectRollback().WillReturnError(dbErr)

	err = repo.RollbackTx(ctx, mockTx)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
}

func TestRollbackTxErrorTxClosed(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectRollback().WillReturnError(pgx.ErrTxClosed)

	err = repo.RollbackTx(ctx, mockTx)

	assert.NoError(t, err)
}

func TestCreateLoanWithSchedule(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	now := time.Now()
	testLoanID := int64(123)

	schedule := []loan.ScheduleEntry{
		{WeekNumber: 1, DueDate: now.AddDate(0, 0, 7), DueAmount: 505.0, Status: loan.PaymentStatus(loan.StatusDelinquent)},
		{WeekNumber: 2, DueDate: now.AddDate(0, 0, 14), DueAmount: 505.0, Status: loan.PaymentStatus(loan.StatusDelinquent)},
	}

	newLoan := &loan.Loan{
		PrincipalAmount:     1000.0,
		InterestRate:        5.0,
		TermWeeks:           2,
		WeeklyPaymentAmount: 505.0,
		TotalLoanAmount:     1010.0,
		StartDate:           now,
		Status:              loan.StatusDelinquent,
		Schedule:            schedule,
	}

	mockPool.ExpectBegin()

	loanSQL := `
        INSERT INTO loans (principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        RETURNING id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at`

	loanRows := pgxmock.NewRows([]string{
		"id", "principal_amount", "interest_rate", "term_weeks", "weekly_payment_amount",
		"total_loan_amount", "start_date", "status", "created_at", "updated_at",
	}).AddRow(
		testLoanID, newLoan.PrincipalAmount, newLoan.InterestRate, newLoan.TermWeeks,
		newLoan.WeeklyPaymentAmount, newLoan.TotalLoanAmount, newLoan.StartDate,
		newLoan.Status, now, now,
	)
	mockPool.ExpectQuery(regexp.QuoteMeta(loanSQL)).
		WithArgs(newLoan.PrincipalAmount, newLoan.InterestRate, newLoan.TermWeeks, newLoan.WeeklyPaymentAmount, newLoan.TotalLoanAmount, newLoan.StartDate, newLoan.Status).
		WillReturnRows(loanRows)

	scheduleSQL := `
            INSERT INTO loan_schedule (loan_id, week_number, due_date, due_amount, status, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, NOW(), NOW())`
	expectBatch := mockPool.ExpectBatch()
	batch := &pgx.Batch{}

	for _, entry := range schedule {
		expectBatch.ExpectExec(regexp.QuoteMeta(scheduleSQL)).
			WithArgs(testLoanID, entry.WeekNumber, entry.DueDate, entry.DueAmount, entry.Status).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		batch.Queue(regexp.QuoteMeta(scheduleSQL), testLoanID, entry.WeekNumber, entry.DueDate, entry.DueAmount, entry.Status)
	}

	mockPool.SendBatch(ctx, batch)
	updateCustomerSQL := `
        UPDATE customers
        SET loan_id = $1, updated_at = NOW()
        WHERE id = $2 AND loan_id IS NULL`
	mockPool.ExpectExec(regexp.QuoteMeta(updateCustomerSQL)).WithArgs(testLoanID, int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mockPool.ExpectCommit()

	customerID := int64(1)

	createdLoan, err := repo.CreateLoan(ctx, customerID, newLoan, schedule)

	assert.NoError(t, err)
	require.NotNil(t, createdLoan)
	assert.Equal(t, testLoanID, createdLoan.ID)
	assert.Equal(t, len(newLoan.Schedule), len(schedule))
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestLoanRepositoryCreateLoanSuccessNoSchedule(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	now := time.Now()
	testLoanID := int64(124)
	customerID := int64(1)
	newLoan := &loan.Loan{
		PrincipalAmount:     2000.0,
		InterestRate:        4.0,
		TermWeeks:           5,
		WeeklyPaymentAmount: 410.0,
		TotalLoanAmount:     2050.0,
		StartDate:           now,
		Status:              loan.StatusActive,
	}
	var schedule []loan.ScheduleEntry

	mockPool.ExpectBegin()

	loanSQL := `
        INSERT INTO loans (principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        RETURNING id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at`

	loanRows := pgxmock.NewRows([]string{
		"id", "principal_amount", "interest_rate", "term_weeks", "weekly_payment_amount",
		"total_loan_amount", "start_date", "status", "created_at", "updated_at",
	}).AddRow(
		testLoanID, newLoan.PrincipalAmount, newLoan.InterestRate, newLoan.TermWeeks,
		newLoan.WeeklyPaymentAmount, newLoan.TotalLoanAmount, newLoan.StartDate,
		newLoan.Status, now, now,
	)
	mockPool.ExpectQuery(regexp.QuoteMeta(loanSQL)).
		WithArgs(newLoan.PrincipalAmount, newLoan.InterestRate, newLoan.TermWeeks, newLoan.WeeklyPaymentAmount, newLoan.TotalLoanAmount, newLoan.StartDate, newLoan.Status).
		WillReturnRows(loanRows)

	updateCustomerSQL := `
        UPDATE customers
        SET loan_id = $1, updated_at = NOW()
        WHERE id = $2 AND loan_id IS NULL`
	mockPool.ExpectExec(regexp.QuoteMeta(updateCustomerSQL)).WithArgs(testLoanID, int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mockPool.ExpectCommit()

	createdLoan, err := repo.CreateLoan(ctx, customerID, newLoan, schedule)

	assert.NoError(t, err)
	require.NotNil(t, createdLoan)
	assert.Equal(t, testLoanID, createdLoan.ID)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestCreateLoanErrorLoanInsertFails(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	newLoan := &loan.Loan{ /* ... minimal setup ... */ }
	var schedule []loan.ScheduleEntry

	dbErr := errors.New("failed to insert loan")

	mockPool.ExpectBegin()
	loanSQL := `
        INSERT INTO loans (principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        RETURNING id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at`
	mockPool.ExpectQuery(regexp.QuoteMeta(loanSQL)).
		WithArgs(newLoan.PrincipalAmount, newLoan.InterestRate, newLoan.TermWeeks, newLoan.WeeklyPaymentAmount, newLoan.TotalLoanAmount, newLoan.StartDate, newLoan.Status).
		WillReturnError(dbErr)

	mockPool.ExpectRollback()

	customerID := int64(1)

	createdLoan, err := repo.CreateLoan(ctx, customerID, newLoan, schedule)

	assert.Error(t, err)
	assert.Nil(t, createdLoan)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
	mockPool.MatchExpectationsInOrder(false)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestLoanRepositoryGetLoanByIDSuccess(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	repo := NewLoanRepository(mockDB, logger)

	ctx := context.Background()
	loanID := int64(1)
	now := time.Now()
	expectedLoan := loan.Loan{
		ID:                  loanID,
		PrincipalAmount:     1000.0,
		InterestRate:        5.0,
		TermWeeks:           10,
		WeeklyPaymentAmount: 105.0,
		TotalLoanAmount:     1050.0,
		StartDate:           now,
		Status:              "PENDING",
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	query := `
        SELECT id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at
        FROM loans
        WHERE id = $1`
	rows := pgxmock.NewRows([]string{
		"id", "principal_amount", "interest_rate", "term_weeks", "weekly_payment_amount",
		"total_loan_amount", "start_date", "status", "created_at", "updated_at",
	}).AddRow(
		expectedLoan.ID, expectedLoan.PrincipalAmount, expectedLoan.InterestRate, expectedLoan.TermWeeks,
		expectedLoan.WeeklyPaymentAmount, expectedLoan.TotalLoanAmount, expectedLoan.StartDate,
		expectedLoan.Status, expectedLoan.CreatedAt, expectedLoan.UpdatedAt,
	)

	mockDB.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	resultLoan, err := repo.GetLoanByID(ctx, loanID)

	assert.NoError(t, err)
	assert.NotNil(t, resultLoan)
	assert.Equal(t, expectedLoan, *resultLoan)

	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestLoanRepositoryGetLoanByIDNotFound(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(999)

	query := `
        SELECT id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at
        FROM loans
        WHERE id = $1`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(loanID).
		WillReturnError(pgx.ErrNoRows)

	resultLoan, err := repo.GetLoanByID(ctx, loanID)

	assert.Error(t, err)
	assert.Nil(t, resultLoan)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetLoanByIDDBError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)
	dbErr := errors.New("connection failure")

	query := `
        SELECT id, principal_amount, interest_rate, term_weeks, weekly_payment_amount, total_loan_amount, start_date, status, created_at, updated_at
        FROM loans
        WHERE id = $1`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(loanID).
		WillReturnError(dbErr)

	resultLoan, err := repo.GetLoanByID(ctx, loanID)

	assert.Error(t, err)
	assert.Nil(t, resultLoan)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetScheduleByLoanIDSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	loanID := int64(1)
	now := time.Now()
	expectedSchedule := []loan.ScheduleEntry{
		{ID: 1, LoanID: loanID, WeekNumber: 1, DueDate: now.AddDate(0, 0, 7), DueAmount: 105.0, PaidAmount: float64(pgtype.Null), PaymentDate: nil, Status: loan.PaymentStatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: 2, LoanID: loanID, WeekNumber: 2, DueDate: now.AddDate(0, 0, 14), DueAmount: 105.0, PaidAmount: float64(pgtype.Null), PaymentDate: nil, Status: loan.PaymentStatusPending, CreatedAt: now, UpdatedAt: now},
	}

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1
        ORDER BY week_number ASC`

	cols := []string{"id", "loan_id", "week_number", "due_date", "due_amount", "paid_amount", "payment_date", "status", "created_at", "updated_at"}
	rows := pgxmock.NewRows(cols)
	for _, entry := range expectedSchedule {
		rows.AddRow(entry.ID, entry.LoanID, entry.WeekNumber, entry.DueDate, entry.DueAmount, entry.PaidAmount, entry.PaymentDate, entry.Status, entry.CreatedAt, entry.UpdatedAt)
	}

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	schedule, err := repo.GetScheduleByLoanID(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, schedule)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetScheduleByLoanIDSuccessEmpty(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(2)

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1
        ORDER BY week_number ASC`

	cols := []string{"id", "loan_id", "week_number", "due_date", "due_amount", "paid_amount", "payment_date", "status", "created_at", "updated_at"}
	rows := pgxmock.NewRows(cols)

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	schedule, err := repo.GetScheduleByLoanID(ctx, loanID)

	assert.NoError(t, err)
	assert.Empty(t, schedule)
	assert.NotNil(t, schedule)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetScheduleByLoanIDDBError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)
	dbErr := errors.New("schedule query failed")

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1
        ORDER BY week_number ASC`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnError(dbErr)

	schedule, err := repo.GetScheduleByLoanID(ctx, loanID)

	assert.Error(t, err)
	assert.Nil(t, schedule)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetUnpaidSchedulesSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	loanID := int64(1)
	now := time.Now()

	expectedSchedule := []loan.ScheduleEntry{
		{ID: 2, LoanID: loanID, WeekNumber: 2, DueDate: now.AddDate(0, 0, 14), DueAmount: 105.0, PaidAmount: float64(pgtype.Null), PaymentDate: nil, Status: loan.PaymentStatusPending, CreatedAt: now, UpdatedAt: now},
	}

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID'
        ORDER BY due_date ASC`

	cols := []string{"id", "loan_id", "week_number", "due_date", "due_amount", "paid_amount", "payment_date", "status", "created_at", "updated_at"}
	rows := pgxmock.NewRows(cols)
	for _, entry := range expectedSchedule {
		rows.AddRow(entry.ID, entry.LoanID, entry.WeekNumber, entry.DueDate, entry.DueAmount, entry.PaidAmount, entry.PaymentDate, entry.Status, entry.CreatedAt, entry.UpdatedAt)
	}

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	schedule, err := repo.GetUnpaidSchedules(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, schedule)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestGetLastTwoDueUnpaidSchedulesSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	loanID := int64(1)
	now := time.Now()

	expectedSchedule := []loan.ScheduleEntry{
		{ID: 4, LoanID: loanID, WeekNumber: 4, DueDate: now.AddDate(0, 0, 28), DueAmount: 105.0, PaidAmount: float64(pgtype.Null), PaymentDate: nil, Status: loan.PaymentStatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: 3, LoanID: loanID, WeekNumber: 3, DueDate: now.AddDate(0, 0, 21), DueAmount: 105.0, PaidAmount: float64(pgtype.Null), PaymentDate: nil, Status: loan.PaymentStatusMissed, CreatedAt: now, UpdatedAt: now},
	}

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status not in ('PAID')
		AND due_date < NOW()
        ORDER BY due_date DESC
        LIMIT 2`

	cols := []string{"id", "loan_id", "week_number", "due_date", "due_amount", "paid_amount", "payment_date", "status", "created_at", "updated_at"}
	rows := pgxmock.NewRows(cols)
	for _, entry := range expectedSchedule {
		rows.AddRow(entry.ID, entry.LoanID, entry.WeekNumber, entry.DueDate, entry.DueAmount, entry.PaidAmount, entry.PaymentDate, entry.Status, entry.CreatedAt, entry.UpdatedAt)
	}

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	schedule, err := repo.GetLastTwoDueUnpaidSchedules(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, schedule)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryFindOldestUnpaidEntryForUpdateSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	loanID := int64(1)
	now := time.Now()
	expectedEntry := loan.ScheduleEntry{
		ID: 1, LoanID: loanID, WeekNumber: 1, DueDate: now.AddDate(0, 0, 7), DueAmount: 105.0, Status: loan.PaymentStatusPending, CreatedAt: now, UpdatedAt: now,
		PaidAmount:  float64(pgtype.Null),
		PaymentDate: nil,
	}

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status = 'PENDING'
        ORDER BY due_date ASC
        LIMIT 1
        FOR UPDATE`

	cols := []string{"id", "loan_id", "week_number", "due_date", "due_amount", "paid_amount", "payment_date", "status", "created_at", "updated_at"}
	rows := pgxmock.NewRows(cols).AddRow(
		expectedEntry.ID, expectedEntry.LoanID, expectedEntry.WeekNumber, expectedEntry.DueDate,
		expectedEntry.DueAmount, expectedEntry.PaidAmount, expectedEntry.PaymentDate,
		expectedEntry.Status, expectedEntry.CreatedAt, expectedEntry.UpdatedAt,
	)

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	entry, err := repo.FindOldestUnpaidEntryForUpdate(ctx, mockPool, loanID)

	assert.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, expectedEntry, *entry)

}

func TestLoanRepositoryFindOldestUnpaidEntryForUpdateNotFound(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)

	query := `
        SELECT id, loan_id, week_number, due_date, due_amount, paid_amount, payment_date, status, created_at, updated_at
        FROM loan_schedule
        WHERE loan_id = $1 AND status = 'PENDING'
        ORDER BY due_date ASC
        LIMIT 1
        FOR UPDATE`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(loanID).
		WillReturnError(pgx.ErrNoRows)

	entry, err := repo.FindOldestUnpaidEntryForUpdate(ctx, mockPool, loanID)

	assert.Error(t, err)
	assert.Nil(t, entry)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestUpdateScheduleEntryInTxSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	now := time.Now()
	entryToUpdate := &loan.ScheduleEntry{
		ID:          1,
		LoanID:      10,
		PaidAmount:  float64(105),
		PaymentDate: &now,
		Status:      loan.PaymentStatusPaid,
	}

	sql := `
        UPDATE loan_schedule
        SET paid_amount = $1, payment_date = $2, status = $3, updated_at = NOW()
        WHERE id = $4 AND loan_id = $5`

	mockPool.ExpectExec(regexp.QuoteMeta(sql)).
		WithArgs(entryToUpdate.PaidAmount, entryToUpdate.PaymentDate, entryToUpdate.Status, entryToUpdate.ID, entryToUpdate.LoanID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateScheduleEntryInTx(ctx, mockPool, entryToUpdate)

	assert.NoError(t, err)
}

func TestLoanRepositoryUpdateScheduleEntryInTxErrorDB(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	entryToUpdate := &loan.ScheduleEntry{ID: 1, LoanID: 10 /* ... */}
	dbErr := errors.New("update failed")

	sql := `
        UPDATE loan_schedule
        SET paid_amount = $1, payment_date = $2, status = $3, updated_at = NOW()
        WHERE id = $4 AND loan_id = $5`

	mockPool.ExpectExec(regexp.QuoteMeta(sql)).
		WithArgs(entryToUpdate.PaidAmount, entryToUpdate.PaymentDate, entryToUpdate.Status, entryToUpdate.ID, entryToUpdate.LoanID).
		WillReturnError(dbErr)

	err := repo.UpdateScheduleEntryInTx(ctx, mockPool, entryToUpdate)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
}

func TestLoanRepositoryUpdateScheduleEntryInTxErrorRowsAffectedZero(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	entryToUpdate := &loan.ScheduleEntry{ID: 1, LoanID: 10 /* ... */}

	sql := `
        UPDATE loan_schedule
        SET paid_amount = $1, payment_date = $2, status = $3, updated_at = NOW()
        WHERE id = $4 AND loan_id = $5`

	mockPool.ExpectExec(regexp.QuoteMeta(sql)).
		WithArgs(entryToUpdate.PaidAmount, entryToUpdate.PaymentDate, entryToUpdate.Status, entryToUpdate.ID, entryToUpdate.LoanID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err := repo.UpdateScheduleEntryInTx(ctx, mockPool, entryToUpdate)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, "affected zero rows")
}

func TestLoanRepositoryUpdateLoanStatusInTxSuccess(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()

	loanID := int64(10)
	newStatus := loan.StatusPaidOff

	sql := `UPDATE loans SET status = $1, updated_at = NOW() WHERE id = $2`

	mockPool.ExpectExec(regexp.QuoteMeta(sql)).
		WithArgs(newStatus, loanID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.UpdateLoanStatusInTx(ctx, mockPool, loanID, newStatus)

	assert.NoError(t, err)
}

func TestCheckIfAllPaymentsMadeInTxTrue(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(10)

	query := `SELECT COUNT(*) FROM loan_schedule WHERE loan_id = $1 AND status != 'PAID'`
	rows := pgxmock.NewRows([]string{"count"}).AddRow(0)

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	allPaid, err := repo.CheckIfAllPaymentsMadeInTx(ctx, mockPool, loanID)

	assert.NoError(t, err)
	assert.True(t, allPaid)
}

func TestLoanRepositoryCheckIfAllPaymentsMadeInTxFalse(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(10)

	query := `SELECT COUNT(*) FROM loan_schedule WHERE loan_id = $1 AND status != 'PAID'`
	rows := pgxmock.NewRows([]string{"count"}).AddRow(2)

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	allPaid, err := repo.CheckIfAllPaymentsMadeInTx(ctx, mockPool, loanID)

	assert.NoError(t, err)
	assert.False(t, allPaid)
}

func TestLoanRepositoryCheckIfAllPaymentsMadeInTxDBError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(10)
	dbErr := errors.New("count query failed")

	query := `SELECT COUNT(*) FROM loan_schedule WHERE loan_id = $1 AND status != 'PAID'`
	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnError(dbErr)

	allPaid, err := repo.CheckIfAllPaymentsMadeInTx(ctx, mockPool, loanID)

	assert.Error(t, err)
	assert.False(t, allPaid)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
}

func TestLoanRepositoryGetTotalOutstandingAmountSuccessPositive(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)
	expectedAmount := 210.50

	query := `
        SELECT COALESCE(SUM(due_amount - paid_amount), 0.00)
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID'`

	rows := pgxmock.NewRows([]string{"coalesce"}).AddRow(expectedAmount)
	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	amount, err := repo.GetTotalOutstandingAmount(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedAmount, amount)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetTotalOutstandingAmountSuccessZero(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)
	expectedAmount := 0.00

	query := `
        SELECT COALESCE(SUM(due_amount - paid_amount), 0.00)
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID'`

	rows := pgxmock.NewRows([]string{"coalesce"}).AddRow(expectedAmount)
	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	amount, err := repo.GetTotalOutstandingAmount(ctx, loanID)
	assert.NoError(t, err)
	assert.Equal(t, expectedAmount, amount)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetTotalOutstandingAmountSuccessNoRowsIsNull(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)

	query := `
		SELECT COALESCE(SUM(due_amount - paid_amount), 0.00)
		FROM loan_schedule
		WHERE loan_id = $1 AND status != 'PAID'`

	rows := pgxmock.NewRows([]string{"coalesce"})
	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	amount, err := repo.GetTotalOutstandingAmount(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, 0.00, amount)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetTotalOutstandingAmountNegativeSumReturnsZero(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)
	dbReturnedAmount := -50.25

	query := `
        SELECT COALESCE(SUM(due_amount - paid_amount), 0.00)
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID'`

	rows := pgxmock.NewRows([]string{"coalesce"}).AddRow(dbReturnedAmount)
	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnRows(rows)

	amount, err := repo.GetTotalOutstandingAmount(ctx, loanID)

	assert.NoError(t, err)

	assert.Equal(t, 0.00, amount)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryGetTotalOutstandingAmountDBError(t *testing.T) {
	ctx, repo, mockPool := setupLoanRepo(t)
	defer mockPool.Close()
	loanID := int64(1)
	dbErr := errors.New("sum query failed")

	query := `
        SELECT COALESCE(SUM(due_amount - paid_amount), 0.00)
        FROM loan_schedule
        WHERE loan_id = $1 AND status != 'PAID'`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(loanID).WillReturnError(dbErr)

	amount, err := repo.GetTotalOutstandingAmount(ctx, loanID)

	assert.Error(t, err)
	assert.Equal(t, 0.00, amount)
	assert.ErrorIs(t, err, apperrors.ErrDatabase)
	assert.ErrorContains(t, err, dbErr.Error())
	assert.NoError(t, mockPool.ExpectationsWereMet())
}
