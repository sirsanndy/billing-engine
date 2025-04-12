package postgres

import (
	"billing-engine/internal/domain/customer"
	"context"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

// type CustomerRepository interface {
// 	Save(ctx context.Context, customer *Customer) error

// 	FindByID(ctx context.Context, customerID int64) (*Customer, error)

// 	FindByLoanID(ctx context.Context, loanID int64) (*Customer, error)

// 	FindAll(ctx context.Context, activeOnly bool) ([]*Customer, error)

// 	Delete(ctx context.Context, customerID int64) error

// 	SetDelinquencyStatus(ctx context.Context, customerID int64, isDelinquent bool) error

// 	SetActiveStatus(ctx context.Context, customerID int64, isActive bool) error
// }

var loanID int64 = int64(123)

var customerTest *customer.Customer = &customer.Customer{
	CustomerID:   1,
	Name:         "John Doe",
	Address:      "123 Main St",
	LoanID:       &loanID,
	Active:       true,
	IsDelinquent: false,
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

func TestCreateCustomerWhenSuccess(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()

	query := `
	INSERT INTO customers (name, address, is_delinquent, active, loan_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	RETURNING id, created_at, updated_at`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(
		customerTest.Name,
		customerTest.Address,
		customerTest.IsDelinquent,
		customerTest.Active,
		customerTest.LoanID,
	).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "update_at"}).
		AddRow(customerTest.CustomerID, customerTest.CreateDate, customerTest.UpdatedAt))

	err := repo.createCustomer(ctx, customerTest)
	assert.NoError(t, err)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestSaveExistingCustomerWhenSuccess(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()

	query := `
	UPDATE customers
	SET name = $1,
		address = $2,
		is_delinquent = $3,
		active = $4,
		loan_id = $5,
		updated_at = NOW()
	WHERE id = $6`

	mockPool.ExpectExec(regexp.QuoteMeta(query)).WithArgs(
		customerTest.Name,
		customerTest.Address,
		customerTest.IsDelinquent,
		customerTest.Active,
		customerTest.LoanID,
		customerTest.CustomerID,
	).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := repo.Save(ctx, customerTest)
	assert.NoError(t, err)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestSaveNonExistingCustomerWhenSuccess(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()
	customerTest.CustomerID = 0

	query := `
	INSERT INTO customers (name, address, is_delinquent, active, loan_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	RETURNING id, created_at, updated_at`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(
		customerTest.Name,
		customerTest.Address,
		customerTest.IsDelinquent,
		customerTest.Active,
		customerTest.LoanID,
	).WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "update_at"}).
		AddRow(customerTest.CustomerID, customerTest.CreateDate, customerTest.UpdatedAt))

	err := repo.Save(ctx, customerTest)
	assert.NoError(t, err)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestFindCustomerByIDReturnOne(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()

	query := `
	SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
	FROM customers
	WHERE id = $1`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(customerTest.CustomerID).WillReturnRows(pgxmock.NewRows([]string{"id", "name", "address", "is_delinquent", "active", "loan_id", "created_at", "updated_at"}).
		AddRow(customerTest.CustomerID, customerTest.Name, customerTest.Address, customerTest.IsDelinquent, customerTest.Active, customerTest.LoanID, customerTest.CreateDate, customerTest.UpdatedAt))

	customerResult, err := repo.FindByID(ctx, customerTest.CustomerID)
	assert.NoError(t, err)
	assert.Equal(t, customerTest.CustomerID, customerResult.CustomerID)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestFindCustomerByIDReturnNone(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()

	query := `
	SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
	FROM customers
	WHERE id = $1`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(customerTest.CustomerID).WillReturnError(pgx.ErrNoRows)

	customerResult, err := repo.FindByID(ctx, customerTest.CustomerID)
	assert.Error(t, err)
	assert.Nil(t, customerResult)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestFindCustomerByLoanIDReturnOne(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()
	query := `
	SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
	FROM customers
	WHERE loan_id = $1`

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(customerTest.LoanID).WillReturnRows(pgxmock.NewRows([]string{"id", "name", "address", "is_delinquent", "active", "loan_id", "created_at", "updated_at"}).
		AddRow(customerTest.CustomerID, customerTest.Name, customerTest.Address, customerTest.IsDelinquent, customerTest.Active, customerTest.LoanID, customerTest.CreateDate, customerTest.UpdatedAt))
	customerResult, err := repo.FindByLoanID(ctx, loanID)
	assert.NoError(t, err)
	assert.Equal(t, customerTest.CustomerID, customerResult.CustomerID)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}

func TestFindAllThenGetAllCustomer(t *testing.T) {
	ctx, repo, mockPool := setupCustomerRepo(t)
	defer mockPool.Close()

	query := `
	SELECT id, name, address, is_delinquent, active, loan_id, created_at, updated_at
	FROM customers WHERE active = $1`
	args := []any{}
	args = append(args, true)
	query += " ORDER BY id ASC"

	mockPool.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "address", "is_delinquent", "active", "loan_id", "created_at", "updated_at"}).
			AddRow(customerTest.CustomerID, customerTest.Name, customerTest.Address, customerTest.IsDelinquent, customerTest.Active, customerTest.LoanID, customerTest.CreateDate, customerTest.UpdatedAt))

	customerResult, err := repo.FindAll(ctx, true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(customerResult))
	assert.Equal(t, customerTest.CustomerID, customerResult[0].CustomerID)
	assert.NoError(t, mockPool.ExpectationsWereMet(), pgxmockExpectationsNotMetMsg)
}
