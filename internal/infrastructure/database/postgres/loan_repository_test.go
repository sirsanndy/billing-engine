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

func (m *MockDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

var logger = slog.New(slog.NewTextHandler(io.Discard, nil))

func setup(t *testing.T) (context.Context, *LoanRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err, "Failed to create mock pool")
	repo := NewLoanRepository(mockPool, logger)
	ctx := context.Background()

	return ctx, repo, mockPool
}

func TestLoanRepositoryBeginTxSuccess(t *testing.T) {
	ctx, repo, mockPool := setup(t)
	defer mockPool.Close()

	mockPool.ExpectBegin()

	tx, err := repo.BeginTx(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestLoanRepositoryBeginTxError(t *testing.T) {
	ctx, repo, mockPool := setup(t)
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
	ctx, repo, mockPool := setup(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectCommit()

	err = repo.CommitTx(ctx, mockTx)

	assert.NoError(t, err)

}

func TestLoanRepositoryCommitTxError(t *testing.T) {
	ctx, repo, mockPool := setup(t)
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
	ctx, repo, mockPool := setup(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectRollback()

	err = repo.RollbackTx(ctx, mockTx)

	assert.NoError(t, err)
}

func TestLoanRepositoryRollbackTxError(t *testing.T) {
	ctx, repo, mockPool := setup(t)
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
	ctx, repo, mockPool := setup(t)
	defer mockPool.Close()
	mockPool.ExpectBegin()
	mockTx, err := mockPool.Begin(ctx)
	require.NoError(t, err)

	mockTx.(pgxmock.PgxCommonIface).ExpectRollback().WillReturnError(pgx.ErrTxClosed)

	err = repo.RollbackTx(ctx, mockTx)

	assert.NoError(t, err)
}

func TestCreateLoanWithSchedule(t *testing.T) {
	ctx, repo, mockPool := setup(t)
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
	mockPool.ExpectCommit()

	createdLoan, err := repo.CreateLoan(ctx, newLoan, schedule)

	assert.NoError(t, err)
	require.NotNil(t, createdLoan)
	assert.Equal(t, testLoanID, createdLoan.ID)
	assert.Equal(t, len(newLoan.Schedule), len(schedule))
	assert.Equal(t, newLoan.PrincipalAmount, createdLoan.PrincipalAmount)

	assert.NoError(t, mockPool.ExpectationsWereMet(), "pgxmock expectations not met")
}

func TestLoanRepositoryGetLoanByID(t *testing.T) {
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
