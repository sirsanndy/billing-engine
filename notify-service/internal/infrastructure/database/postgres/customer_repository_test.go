package postgres

import (
	"context"
	"io"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"notify-service/internal/domain/customer"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var logger = slog.New(slog.NewTextHandler(io.Discard, nil))

const pgxmockExpectationsNotMetMsg = "pgxmock expectations not met"

type MockPgxPool struct {
	mock.Mock
}

func setupCustomerRepo(t *testing.T) (context.Context, *CustomerRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to open a stub database connection: %v", err)
	}

	ctx := context.Background()
	repo := NewCustomerRepository(mockPool, logger)

	return ctx, repo, mockPool
}
func TestCustomerRepositoryUpsert(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()
	var loanID int64 = int64(123)

	var customerTest *customer.Customer = &customer.Customer{
		CustomerID:   1,
		Name:         "John Doe",
		Address:      "123 Main St",
		LoanID:       &loanID,
		Active:       true,
		IsDelinquent: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

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
		WHERE customers.updated_at < EXCLUDED.updated_at
		RETURNING (xmax = 0) AS is_insert;
	`

	t.Run("successful upsert", func(t *testing.T) {
		mockPool.ExpectQuery(regexp.QuoteMeta(upsertSQL)).
			WithArgs(
				customerTest.CustomerID,
				customerTest.Name,
				customerTest.Address,
				customerTest.IsDelinquent,
				customerTest.Active,
				customerTest.LoanID,
				customerTest.CreatedAt,
				customerTest.UpdatedAt,
			).WillReturnRows(pgxmock.NewRows([]string{"is_insert"}).
			AddRow(true))

		err := repo.Upsert(ctx, customerTest)
		assert.NoError(t, err)
		assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
	})

	t.Run("failed upsert", func(t *testing.T) {
		mockPool.ExpectQuery(regexp.QuoteMeta(upsertSQL)).WithArgs(
			customerTest.CustomerID,
			customerTest.Name,
			customerTest.Address,
			customerTest.IsDelinquent,
			customerTest.Active,
			customerTest.LoanID,
			customerTest.CreatedAt,
			customerTest.UpdatedAt,
		).WillReturnError(context.DeadlineExceeded)

		err := repo.Upsert(ctx, customerTest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upsert customer 1: context deadline exceeded")
		assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
	})
}
