package postgres

import (
	"billing-engine/internal/domain/loan"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDB struct {
	mock.Mock
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

func (m *MockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgx.CommandTag, error) {
	argsMock := m.Called(ctx, sql, args)
	return argsMock.Get(0).(pgx.CommandTag), argsMock.Error(1)
}

func TestLoanRepository_CreateLoan(t *testing.T) {
	mockDB := new(MockDB)
	logger := slog.New(slog.NewTextHandler(nil))
	repo := NewLoanRepository(mockDB, logger)

	ctx := context.Background()
	newLoan := &loan.Loan{
		PrincipalAmount:     1000.0,
		InterestRate:        5.0,
		TermWeeks:           10,
		WeeklyPaymentAmount: 105.0,
		TotalLoanAmount:     1050.0,
		StartDate:           time.Now(),
		Status:              "PENDING",
	}
	schedule := []loan.ScheduleEntry{
		{WeekNumber: 1, DueDate: time.Now().AddDate(0, 0, 7), DueAmount: 105.0, Status: "PENDING"},
	}

	mockTx := new(MockDB)
	mockDB.On("Begin", ctx).Return(mockTx, nil)
	mockTx.On("QueryRow", ctx, mock.Anything, mock.Anything).Return(mockTx)
	mockTx.On("Exec", ctx, mock.Anything, mock.Anything).Return(pgx.CommandTag{}, nil)
	mockTx.On("Commit", ctx).Return(nil)

	createdLoan, err := repo.CreateLoan(ctx, newLoan, schedule)

	assert.NoError(t, err)
	assert.NotNil(t, createdLoan)
	mockDB.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestLoanRepository_GetLoanByID(t *testing.T) {
	mockDB := new(MockDB)
	logger := slog.New(slog.NewTextHandler(nil))
	repo := NewLoanRepository(mockDB, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedLoan := &loan.Loan{
		ID:                  loanID,
		PrincipalAmount:     1000.0,
		InterestRate:        5.0,
		TermWeeks:           10,
		WeeklyPaymentAmount: 105.0,
		TotalLoanAmount:     1050.0,
		StartDate:           time.Now(),
		Status:              "PENDING",
	}

	mockRow := new(MockDB)
	mockDB.On("QueryRow", ctx, mock.Anything, loanID).Return(mockRow)
	mockRow.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		loan := args.Get(0).(*loan.Loan)
		*loan = *expectedLoan
	}).Return(nil)

	loan, err := repo.GetLoanByID(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedLoan, loan)
	mockDB.AssertExpectations(t)
	mockRow.AssertExpectations(t)
}

func TestLoanRepository_GetScheduleByLoanID(t *testing.T) {
	mockDB := new(MockDB)
	logger := slog.New(slog.NewTextHandler(nil))
	repo := NewLoanRepository(mockDB, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedSchedule := []loan.ScheduleEntry{
		{ID: 1, LoanID: loanID, WeekNumber: 1, DueDate: time.Now().AddDate(0, 0, 7), DueAmount: 105.0, Status: "PENDING"},
		{ID: 2, LoanID: loanID, WeekNumber: 2, DueDate: time.Now().AddDate(0, 0, 14), DueAmount: 105.0, Status: "PENDING"},
	}

	mockRows := new(MockDB)
	mockDB.On("Query", ctx, mock.Anything, loanID).Return(mockRows, nil)
	mockRows.On("Next").Return(true).Times(len(expectedSchedule))
	mockRows.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
		entry := args.Get(0).(*loan.ScheduleEntry)
		*entry = expectedSchedule[0]
		expectedSchedule = expectedSchedule[1:]
	}).Return(nil)
	mockRows.On("Err").Return(nil)
	mockRows.On("Close").Return(nil)

	schedule, err := repo.GetScheduleByLoanID(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, schedule)
	mockDB.AssertExpectations(t)
	mockRows.AssertExpectations(t)
}
