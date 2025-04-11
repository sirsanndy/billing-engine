package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"billing-engine/internal/domain/customer"
	"billing-engine/internal/pkg/apperrors"

	"github.com/jackc/pgx/v5"
)

type CustomerRepository struct {
	db     DBPool
	logger *slog.Logger
}

var _ customer.CustomerRepository = (*CustomerRepository)(nil)

func NewCustomerRepository(db DBPool, logger *slog.Logger) *CustomerRepository {
	if db == nil {
		panic("DBPool cannot be nil for CustomerRepository")
	}
	if logger == nil {

		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		logger.Warn("Warning: No logger provided to NewCustomerRepository, using default stderr handler")
	}
	return &CustomerRepository{
		db:     db,
		logger: logger.With("component", "CustomerRepository"),
	}
}

func (r *CustomerRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {

	r.logger.InfoContext(ctx, "Beginning transaction")
	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to begin transaction", slog.Any("error", err))

		return nil, fmt.Errorf("%w: failed to begin transaction: %w", apperrors.ErrDatabase, err)
	}
	return tx, nil
}

func (r *CustomerRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {

	r.logger.InfoContext(ctx, "Committing transaction")
	err := tx.Commit(ctx)
	if err != nil {

		r.logger.ErrorContext(ctx, "Failed to commit transaction", slog.Any("error", err))
		return fmt.Errorf("%w: failed to commit transaction: %w", apperrors.ErrDatabase, err)
	}
	r.logger.InfoContext(ctx, "Transaction committed successfully")
	return nil
}

func (r *CustomerRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {

	r.logger.InfoContext(ctx, "Rolling back transaction")
	err := tx.Rollback(ctx)

	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		r.logger.ErrorContext(ctx, "Failed to rollback transaction", slog.Any("error", err))
		return fmt.Errorf("%w: failed to rollback transaction: %w", apperrors.ErrDatabase, err)
	}
	if err == nil {
		r.logger.InfoContext(ctx, "Transaction rolled back successfully")
	} else {
		r.logger.InfoContext(ctx, "Transaction rollback attempted on closed transaction")
	}
	return nil
}

func (r *CustomerRepository) Save(ctx context.Context, cust *customer.Customer) error {
	if cust == nil {
		return fmt.Errorf("%w: customer cannot be nil", apperrors.ErrInvalidArgument)
	}

	if cust.CustomerID == 0 {

		return r.createCustomer(ctx, cust)
	} else {

		return r.updateCustomer(ctx, cust)
	}
}

func (r *CustomerRepository) createCustomer(ctx context.Context, cust *customer.Customer) error {

	r.logger.InfoContext(ctx, "Attempting to insert new customer", slog.String("name", cust.Name))

	query := `
        INSERT INTO customers (name, address, is_delinquent, active, loan_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
        RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		cust.Name,
		cust.Address,
		cust.IsDelinquent,
		cust.Active,
		cust.LoanID,
	).Scan(
		&cust.CustomerID,
		&cust.CreateDate,
		&cust.UpdatedAt,
	)

	if err != nil {

		translatedErr := translateDBError(err, r.logger)
		if errors.Is(translatedErr, apperrors.ErrAlreadyExists) {

			r.logger.WarnContext(ctx, "Failed to insert customer due to unique constraint violation", cust.LoanID)

			return translatedErr
		}
		r.logger.ErrorContext(ctx, "Failed to insert customer", slog.Any("error", err))
		return fmt.Errorf("%w: failed to insert customer: %w", apperrors.ErrDatabase, err)
	}

	r.logger.InfoContext(ctx, "Customer inserted successfully", slog.Int64("customerID", cust.CustomerID))
	return nil
}

func (r *CustomerRepository) updateCustomer(ctx context.Context, cust *customer.Customer) error {

	r.logger.InfoContext(ctx, "Attempting to update customer")

	query := `
        UPDATE customers
        SET name = $1,
            address = $2,
            is_delinquent = $3,
            active = $4,
            loan_id = $5,
            updated_at = NOW()
        WHERE id = $6`

	cmdTag, err := r.db.Exec(ctx, query,
		cust.Name,
		cust.Address,
		cust.IsDelinquent,
		cust.Active,
		cust.LoanID,
		cust.CustomerID,
	)

	if err != nil {

		translatedErr := translateDBError(err, r.logger)
		if errors.Is(translatedErr, apperrors.ErrAlreadyExists) {
			r.logger.WarnContext(ctx, "Failed to update customer due to unique constraint violation", slog.Any("error", err), cust.LoanID)
			return translatedErr
		}
		r.logger.ErrorContext(ctx, "Failed to update customer", slog.Any("error", err))
		return fmt.Errorf("%w: failed to update customer: %w", apperrors.ErrDatabase, err)
	}

	if cmdTag.RowsAffected() == 0 {
		r.logger.WarnContext(ctx, "Update affected zero rows, customer likely not found")

		return apperrors.ErrNotFound
	}

	r.logger.InfoContext(ctx, "Customer updated successfully")

	return nil
}

func (r *CustomerRepository) FindByID(ctx context.Context, customerID int64) (*customer.Customer, error) {

	r.logger.InfoContext(ctx, "Attempting to find customer by ID")

	query := `
        SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
        FROM customers
        WHERE id = $1`

	var cust customer.Customer
	err := r.db.QueryRow(ctx, query, customerID).Scan(
		&cust.CustomerID,
		&cust.Name,
		&cust.Address,
		&cust.IsDelinquent,
		&cust.Active,
		&cust.LoanID,
		&cust.CreateDate,
		&cust.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WarnContext(ctx, "Customer not found")
			return nil, apperrors.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "Failed to query/scan customer by ID", slog.Any("error", err))
		return nil, fmt.Errorf("%w: failed to get customer by ID: %w", apperrors.ErrDatabase, err)
	}

	r.logger.InfoContext(ctx, "Customer found successfully")
	return &cust, nil
}

func (r *CustomerRepository) FindByLoanID(ctx context.Context, loanID int64) (*customer.Customer, error) {

	r.logger.InfoContext(ctx, "Attempting to find customer by loan ID")

	query := `
        SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
        FROM customers
        WHERE loan_id = $1`

	var cust customer.Customer
	err := r.db.QueryRow(ctx, query, loanID).Scan(
		&cust.CustomerID,
		&cust.Name,
		&cust.Address,
		&cust.IsDelinquent,
		&cust.Active,
		&cust.LoanID,
		&cust.CreateDate,
		&cust.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WarnContext(ctx, "Customer not found for the given loan ID")
			return nil, apperrors.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "Failed to query/scan customer by loan ID", slog.Any("error", err))
		return nil, fmt.Errorf("%w: failed to get customer by loan ID: %w", apperrors.ErrDatabase, err)
	}

	r.logger.InfoContext(ctx, "Customer found successfully by loan ID", slog.Int64("customerID", cust.CustomerID))
	return &cust, nil
}

func (r *CustomerRepository) FindAll(ctx context.Context, activeOnly bool) ([]*customer.Customer, error) {

	r.logger.InfoContext(ctx, "Attempting to find all customers")

	baseQuery := `
        SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
        FROM customers`
	args := []any{}
	query := baseQuery
	if activeOnly {
		query += " WHERE active = $1"
		args = append(args, true)
	}
	query += " ORDER BY id ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {

		r.logger.ErrorContext(ctx, "Failed to query customers", slog.Any("error", err))
		return nil, fmt.Errorf("%w: failed to query customers: %w", apperrors.ErrDatabase, err)
	}
	defer rows.Close()

	customers := make([]*customer.Customer, 0)
	for rows.Next() {
		var cust customer.Customer
		err := rows.Scan(
			&cust.CustomerID,
			&cust.Name,
			&cust.Address,
			&cust.IsDelinquent,
			&cust.Active,
			&cust.LoanID,
			&cust.CreateDate,
			&cust.UpdatedAt,
		)
		if err != nil {
			r.logger.ErrorContext(ctx, "Failed to scan customer row", slog.Any("error", err))

			return nil, fmt.Errorf("%w: failed to scan customer row: %w", apperrors.ErrDatabase, err)
		}
		customers = append(customers, &cust)
	}

	if err = rows.Err(); err != nil {
		r.logger.ErrorContext(ctx, "Error iterating customer rows", slog.Any("error", err))
		return nil, fmt.Errorf("%w: error iterating customer rows: %w", apperrors.ErrDatabase, err)
	}

	r.logger.InfoContext(ctx, "Finished finding customers", slog.Int("count", len(customers)))
	return customers, nil
}

func (r *CustomerRepository) Delete(ctx context.Context, customerID int64) error {

	r.logger.InfoContext(ctx, "Attempting to delete customer")

	query := `DELETE FROM customers WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, customerID)
	if err != nil {

		r.logger.ErrorContext(ctx, "Failed to execute delete customer", slog.Any("error", err))
		return fmt.Errorf("%w: failed to delete customer: %w", apperrors.ErrDatabase, err)
	}

	if cmdTag.RowsAffected() == 0 {
		r.logger.WarnContext(ctx, "Delete affected zero rows, customer likely not found")
		return apperrors.ErrNotFound
	}

	r.logger.InfoContext(ctx, "Customer deleted successfully")
	return nil
}

func (r *CustomerRepository) SetDelinquencyStatus(ctx context.Context, customerID int64, isDelinquent bool) error {
	r.logger.InfoContext(ctx, "Attempting to set delinquency status")

	query := `UPDATE customers SET is_delinquent = $1, updated_at = NOW() WHERE id = $2`

	cmdTag, err := r.db.Exec(ctx, query, isDelinquent, customerID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to execute update delinquency status", slog.Any("error", err))
		return fmt.Errorf("%w: failed to update delinquency status: %w", apperrors.ErrDatabase, err)
	}

	if cmdTag.RowsAffected() == 0 {
		r.logger.WarnContext(ctx, "Update delinquency affected zero rows, customer likely not found")
		return apperrors.ErrNotFound
	}

	r.logger.InfoContext(ctx, "Customer delinquency status updated successfully")
	return nil
}

func (r *CustomerRepository) SetActiveStatus(ctx context.Context, customerID int64, isActive bool) error {

	r.logger.InfoContext(ctx, "Attempting to set active status")

	query := `UPDATE customers SET active = $1, updated_at = NOW() WHERE id = $2`

	cmdTag, err := r.db.Exec(ctx, query, isActive, customerID)
	if err != nil {
		r.logger.ErrorContext(ctx, "Failed to execute update active status", slog.Any("error", err))
		return fmt.Errorf("%w: failed to update active status: %w", apperrors.ErrDatabase, err)
	}

	if cmdTag.RowsAffected() == 0 {
		r.logger.WarnContext(ctx, "Update active status affected zero rows, customer likely not found")
		return apperrors.ErrNotFound
	}

	r.logger.InfoContext(ctx, "Customer active status updated successfully")
	return nil
}
