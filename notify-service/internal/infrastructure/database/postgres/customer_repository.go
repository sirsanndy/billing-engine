package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"notify-service/internal/domain/customer"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v4"
)

type CustomerRepository struct {
	db     DBPool
	logger *slog.Logger
}

var _ customer.CustomerRepository = (*CustomerRepository)(nil)
var _ DBPool = (*pgxpool.Pool)(nil)

var _ DBPool = (pgxmock.PgxPoolIface)(nil)

type DBPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Acquire(ctx context.Context) (*pgxpool.Conn, error)
	Close()
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

func (r *CustomerRepository) Upsert(ctx context.Context, cust *customer.Customer) error {
	r.logger.With(slog.Int64("customerID", cust.CustomerID))
	r.logger.DebugContext(ctx, "Attempting to upsert customer")

	upsertSQL := `
        INSERT INTO customers (id, name, address, is_delinquent, active, loan_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (id) DO UPDATE SET
            name = EXCLUDED.name,
            address = EXCLUDED.address,
            is_delinquent = EXCLUDED.is_delinquent,
            active = EXCLUDED.active,
            loan_id = EXCLUDED.loan_id,
            -- created_at should not be updated on conflict
            updated_at = EXCLUDED.updated_at
        WHERE customers.updated_at < EXCLUDED.updated_at;
        -- Or remove WHERE clause if last write should always win:
        -- updated_at = EXCLUDED.updated_at
    `

	cmdTag, err := r.db.Exec(ctx, upsertSQL,
		cust.CustomerID,
		cust.Name,
		cust.Address,
		cust.IsDelinquent,
		cust.Active,
		cust.LoanID,
		cust.CreatedAt,
		cust.UpdatedAt,
	)

	if err != nil {
		r.logger.ErrorContext(ctx, "Database upsert failed", slog.Any("error", err))
		return fmt.Errorf("failed to upsert customer %d: %w", cust.CustomerID, err)
	}

	r.logger.InfoContext(ctx, "Customer upsert successful", slog.Int64("rows_affected", cmdTag.RowsAffected()))
	return nil
}
