package loan

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateLoan(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.New(slog.NewTextHandler(nil, nil))
	service := NewLoanService(mockRepo, logger)

	ctx := context.Background()
	principal := Money(1000)
	termWeeks := 52
	annualInterestRate := Money(5)
	startDate := time.Now()

	loan := &Loan{}
	mockRepo.On("CreateLoan", ctx, mock.Anything, mock.Anything).Return(loan, nil)

	result, err := service.CreateLoan(ctx, principal, termWeeks, annualInterestRate, startDate)

	assert.NoError(t, err)
	assert.Equal(t, loan, result)
	mockRepo.AssertExpectations(t)
}

func TestGetOutstanding(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.New(slog.NewTextHandler(nil, nil))
	service := NewLoanService(mockRepo, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedOutstanding := Money(500)

	mockRepo.On("GetTotalOutstandingAmount", ctx, loanID).Return(expectedOutstanding, nil)

	result, err := service.GetOutstanding(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedOutstanding, result)
	mockRepo.AssertExpectations(t)
}

func TestIsDelinquent(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.New(slog.NewTextHandler(nil, nil))
	service := NewLoanService(mockRepo, logger)

	ctx := context.Background()
	loanID := int64(1)
	lastTwoUnpaid := []ScheduleEntry{{}, {}}

	mockRepo.On("GetLastTwoDueUnpaidSchedules", ctx, loanID).Return(lastTwoUnpaid, nil)

	result, err := service.IsDelinquent(ctx, loanID)

	assert.NoError(t, err)
	assert.True(t, result)
	mockRepo.AssertExpectations(t)
}

func TestMakePayment(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.New(slog.NewTextHandler(nil, nil))
	service := NewLoanService(mockRepo, logger)

	ctx := context.Background()
	loanID := int64(1)
	amount := Money(100)
	tx := struct{}{}
	entry := &ScheduleEntry{DueAmount: amount}

	mockRepo.On("BeginTx", ctx).Return(tx, nil)
	mockRepo.On("FindOldestUnpaidEntryForUpdate", ctx, tx, loanID).Return(entry, nil)
	mockRepo.On("UpdateScheduleEntryInTx", ctx, tx, entry).Return(nil)
	mockRepo.On("CheckIfAllPaymentsMadeInTx", ctx, tx, loanID).Return(false, nil)
	mockRepo.On("CommitTx", ctx, tx).Return(nil)

	err := service.MakePayment(ctx, loanID, amount)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetLoan(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.New(slog.NewTextHandler(nil, nil))
	service := NewLoanService(mockRepo, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedLoan := &Loan{}

	mockRepo.On("GetLoanByID", ctx, loanID).Return(expectedLoan, nil)

	result, err := service.GetLoan(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedLoan, result)
	mockRepo.AssertExpectations(t)
}

func TestGetLoanSchedule(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := slog.New(slog.NewTextHandler(nil, nil))
	service := NewLoanService(mockRepo, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedSchedule := []ScheduleEntry{{}, {}}

	mockRepo.On("GetScheduleByLoanID", ctx, loanID).Return(expectedSchedule, nil)

	result, err := service.GetLoanSchedule(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, result)
	mockRepo.AssertExpectations(t)
}
