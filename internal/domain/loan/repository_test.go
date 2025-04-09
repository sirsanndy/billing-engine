package loan

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRepository struct {
	mock.Mock
}

type TxMock struct {
	pgx.Tx
}

var tx pgx.Tx = &TxMock{}

func (m *MockRepository) CreateLoan(ctx context.Context, loan *Loan, schedule []ScheduleEntry) (*Loan, error) {
	args := m.Called(ctx, loan, schedule)
	return args.Get(0).(*Loan), args.Error(1)
}

func (m *MockRepository) GetLoanByID(ctx context.Context, loanID int64) (*Loan, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).(*Loan), args.Error(1)
}

func (m *MockRepository) GetScheduleByLoanID(ctx context.Context, loanID int64) ([]ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).([]ScheduleEntry), args.Error(1)
}

func (m *MockRepository) GetUnpaidSchedules(ctx context.Context, loanID int64) ([]ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).([]ScheduleEntry), args.Error(1)
}

func (m *MockRepository) GetLastTwoDueUnpaidSchedules(ctx context.Context, loanID int64) ([]ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).([]ScheduleEntry), args.Error(1)
}

func (m *MockRepository) FindOldestUnpaidEntryForUpdate(ctx context.Context, tx pgx.Tx, loanID int64) (*ScheduleEntry, error) {
	args := m.Called(ctx, tx, loanID)
	return args.Get(0).(*ScheduleEntry), args.Error(1)
}

func (m *MockRepository) UpdateScheduleEntryInTx(ctx context.Context, tx pgx.Tx, entry *ScheduleEntry) error {
	args := m.Called(ctx, tx, entry)
	return args.Error(0)
}

func (m *MockRepository) UpdateLoanStatusInTx(ctx context.Context, tx pgx.Tx, loanID int64, status LoanStatus) error {
	args := m.Called(ctx, tx, loanID, status)
	return args.Error(0)
}

func (m *MockRepository) CheckIfAllPaymentsMadeInTx(ctx context.Context, tx pgx.Tx, loanID int64) (bool, error) {
	args := m.Called(ctx, tx, loanID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetTotalOutstandingAmount(ctx context.Context, loanID int64) (float64, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func TestRepository_CreateLoan(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loan := &Loan{}
	schedule := []ScheduleEntry{}
	expectedLoan := &Loan{}

	mockRepo.On("CreateLoan", ctx, loan, schedule).Return(expectedLoan, nil)

	result, err := mockRepo.CreateLoan(ctx, loan, schedule)
	require.NoError(t, err)
	require.Equal(t, expectedLoan, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_GetLoanByID(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	expectedLoan := &Loan{}

	mockRepo.On("GetLoanByID", ctx, loanID).Return(expectedLoan, nil)

	result, err := mockRepo.GetLoanByID(ctx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedLoan, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_GetScheduleByLoanID(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	expectedSchedule := []ScheduleEntry{}

	mockRepo.On("GetScheduleByLoanID", ctx, loanID).Return(expectedSchedule, nil)

	result, err := mockRepo.GetScheduleByLoanID(ctx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedSchedule, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_GetUnpaidSchedules(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	expectedSchedules := []ScheduleEntry{}

	mockRepo.On("GetUnpaidSchedules", ctx, loanID).Return(expectedSchedules, nil)

	result, err := mockRepo.GetUnpaidSchedules(ctx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedSchedules, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_GetLastTwoDueUnpaidSchedules(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	expectedSchedules := []ScheduleEntry{}

	mockRepo.On("GetLastTwoDueUnpaidSchedules", ctx, loanID).Return(expectedSchedules, nil)

	result, err := mockRepo.GetLastTwoDueUnpaidSchedules(ctx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedSchedules, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_FindOldestUnpaidEntryForUpdate(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	expectedEntry := &ScheduleEntry{}

	mockRepo.On("FindOldestUnpaidEntryForUpdate", ctx, tx, loanID).Return(expectedEntry, nil)

	result, err := mockRepo.FindOldestUnpaidEntryForUpdate(ctx, tx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedEntry, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_UpdateScheduleEntryInTx(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	entry := &ScheduleEntry{}

	mockRepo.On("UpdateScheduleEntryInTx", ctx, tx, entry).Return(nil)

	err := mockRepo.UpdateScheduleEntryInTx(ctx, tx, entry)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestRepository_UpdateLoanStatusInTx(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	status := LoanStatus("PAID")

	mockRepo.On("UpdateLoanStatusInTx", ctx, tx, loanID, status).Return(nil)

	err := mockRepo.UpdateLoanStatusInTx(ctx, tx, loanID, status)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestRepository_CheckIfAllPaymentsMadeInTx(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()

	loanID := int64(1)
	expectedResult := true

	mockRepo.On("CheckIfAllPaymentsMadeInTx", ctx, tx, loanID).Return(expectedResult, nil)

	result, err := mockRepo.CheckIfAllPaymentsMadeInTx(ctx, tx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedResult, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_GetTotalOutstandingAmount(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	loanID := int64(1)
	expectedAmount := 1000.0

	mockRepo.On("GetTotalOutstandingAmount", ctx, loanID).Return(expectedAmount, nil)

	result, err := mockRepo.GetTotalOutstandingAmount(ctx, loanID)
	require.NoError(t, err)
	require.Equal(t, expectedAmount, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_BeginTx(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()

	mockRepo.On("BeginTx", ctx).Return(tx, nil)

	result, err := mockRepo.BeginTx(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	mockRepo.AssertExpectations(t)
}

func TestRepository_CommitTx(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()

	mockRepo.On("CommitTx", ctx, tx).Return(nil)

	err := mockRepo.CommitTx(ctx, tx)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestRepository_RollbackTx(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()

	mockRepo.On("RollbackTx", ctx, tx).Return(nil)

	err := mockRepo.RollbackTx(ctx, tx)
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}
