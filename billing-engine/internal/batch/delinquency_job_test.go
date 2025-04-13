package batch_test

import (
	"billing-engine/internal/batch"
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/pkg/apperrors"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLoanService struct {
	mock.Mock
}

func (m *MockLoanService) GetLoanSchedule(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	if schedule, ok := args.Get(0).([]loan.ScheduleEntry); ok {
		return schedule, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockLoanService) GetLoan(ctx context.Context, loanID int64) (*loan.Loan, error) {
	args := m.Called(ctx, loanID)
	if loan, ok := args.Get(0).(*loan.Loan); ok {
		return loan, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockLoanService) CreateLoan(ctx context.Context, customerID int64, annualInterestRate loan.Money, termWeeks int, amount loan.Money, time time.Time) (*loan.Loan, error) {
	args := m.Called(ctx, customerID, annualInterestRate, termWeeks, amount)
	if createdLoan, ok := args.Get(0).(*loan.Loan); ok {
		return createdLoan, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockLoanService) MakePayment(ctx context.Context, loanID int64, amount loan.Money) error {
	args := m.Called(ctx, loanID, amount)
	return args.Error(0)
}

func (m *MockLoanService) GetOutstanding(ctx context.Context, loanID int64) (loan.Money, error) {
	args := m.Called(ctx, loanID)
	if outstanding, ok := args.Get(0).(loan.Money); ok {
		return outstanding, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *MockLoanService) IsDelinquent(ctx context.Context, loanID int64) (bool, error) {
	args := m.Called(ctx, loanID)
	if isDelinquent, ok := args.Get(0).(bool); ok {
		return isDelinquent, args.Error(1)
	}
	return false, args.Error(1)
}

type MockLoanRepository struct {
	mock.Mock
}

type TxMock struct {
	pgx.Tx
}

var tx pgx.Tx = &TxMock{}

func (m *MockLoanRepository) CreateLoan(ctx context.Context, customerId int64, loanTest *loan.Loan, schedule []loan.ScheduleEntry) (*loan.Loan, error) {
	args := m.Called(ctx, customerId, loanTest, schedule)
	return args.Get(0).(*loan.Loan), args.Error(1)
}

func (m *MockLoanRepository) GetLoanByID(ctx context.Context, loanID int64) (*loan.Loan, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).(*loan.Loan), args.Error(1)
}

func (m *MockLoanRepository) GetScheduleByLoanID(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).([]loan.ScheduleEntry), args.Error(1)
}

func (m *MockLoanRepository) GetUnpaidSchedules(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).([]loan.ScheduleEntry), args.Error(1)
}

func (m *MockLoanRepository) GetLastTwoDueUnpaidSchedules(ctx context.Context, loanID int64) ([]loan.ScheduleEntry, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).([]loan.ScheduleEntry), args.Error(1)
}

func (m *MockLoanRepository) FindOldestUnpaidEntryForUpdate(ctx context.Context, tx pgx.Tx, loanID int64) (*loan.ScheduleEntry, error) {
	args := m.Called(ctx, tx, loanID)
	return args.Get(0).(*loan.ScheduleEntry), args.Error(1)
}

func (m *MockLoanRepository) UpdateScheduleEntryInTx(ctx context.Context, tx pgx.Tx, entry *loan.ScheduleEntry) error {
	args := m.Called(ctx, tx, entry)
	return args.Error(0)
}

func (m *MockLoanRepository) UpdateLoanStatusInTx(ctx context.Context, tx pgx.Tx, loanID int64, status loan.LoanStatus) error {
	args := m.Called(ctx, tx, loanID, status)
	return args.Error(0)
}

func (m *MockLoanRepository) CheckIfAllPaymentsMadeInTx(ctx context.Context, tx pgx.Tx, loanID int64) (bool, error) {
	args := m.Called(ctx, tx, loanID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLoanRepository) GetTotalOutstandingAmount(ctx context.Context, loanID int64) (float64, error) {
	args := m.Called(ctx, loanID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockLoanRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockLoanRepository) CommitTx(ctx context.Context, tx pgx.Tx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockLoanRepository) RollbackTx(ctx context.Context, tx pgx.Tx) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockLoanRepository) GetAllActiveLoanIDs(ctx context.Context) ([]int64, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]int64), args.Error(1)
	}
	return nil, args.Error(1)
}

type MockCustomerService struct {
	mock.Mock
}

func (_m *MockCustomerService) CreateNewCustomer(ctx context.Context, name string, address string) (*customer.Customer, error) {
	ret := _m.Called(ctx, name, address)

	var r0 *customer.Customer
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *customer.Customer); ok {
		r0 = rf(ctx, name, address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*customer.Customer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockCustomerService) GetCustomer(ctx context.Context, customerID int64) (*customer.Customer, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *customer.Customer
	if rf, ok := ret.Get(0).(func(context.Context, int64) *customer.Customer); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*customer.Customer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockCustomerService) ListActiveCustomers(ctx context.Context) ([]*customer.Customer, error) {
	ret := _m.Called(ctx)

	var r0 []*customer.Customer
	if rf, ok := ret.Get(0).(func(context.Context) []*customer.Customer); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*customer.Customer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockCustomerService) UpdateCustomerAddress(ctx context.Context, customerID int64, newAddress string) error {
	ret := _m.Called(ctx, customerID, newAddress)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, string) error); ok {
		r0 = rf(ctx, customerID, newAddress)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerService) AssignLoanToCustomer(ctx context.Context, customerID int64, loanID int64) error {
	ret := _m.Called(ctx, customerID, loanID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, int64) error); ok {
		r0 = rf(ctx, customerID, loanID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerService) UpdateDelinquency(ctx context.Context, customerID int64, isDelinquent bool) error {
	ret := _m.Called(ctx, customerID, isDelinquent)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, bool) error); ok {
		r0 = rf(ctx, customerID, isDelinquent)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerService) DeactivateCustomer(ctx context.Context, customerID int64) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerService) ReactivateCustomer(ctx context.Context, customerID int64) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerService) FindCustomerByLoan(ctx context.Context, loanID int64) (*customer.Customer, error) {
	ret := _m.Called(ctx, loanID)

	var r0 *customer.Customer
	if rf, ok := ret.Get(0).(func(context.Context, int64) *customer.Customer); ok {
		r0 = rf(ctx, loanID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*customer.Customer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, loanID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func TestUpdateDelinquencyJobRun(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("successfully processes loans", func(t *testing.T) {
		activeLoanIDs := []int64{1, 2}
		mockLoanRepo, mockLoanService, mockCustomerService, job := newFunction(logger)
		mockLoanRepo.On("GetAllActiveLoanIDs", ctx).Return(activeLoanIDs, nil)

		mockLoanService.On("IsDelinquent", ctx, int64(1)).Return(true, nil)
		mockLoanService.On("IsDelinquent", ctx, int64(2)).Return(false, nil)

		mockCustomerService.On("FindCustomerByLoan", ctx, int64(1)).Return(&customer.Customer{CustomerID: 101, IsDelinquent: false}, nil)
		mockCustomerService.On("FindCustomerByLoan", ctx, int64(2)).Return(&customer.Customer{CustomerID: 102, IsDelinquent: true}, nil)

		mockCustomerService.On("UpdateDelinquency", ctx, int64(101), true).Return(nil)
		mockCustomerService.On("UpdateDelinquency", ctx, int64(102), false).Return(nil)

		err := job.Run(ctx)
		assert.NoError(t, err)

		mockLoanRepo.AssertExpectations(t)
		mockLoanService.AssertExpectations(t)
		mockCustomerService.AssertExpectations(t)
	})

	t.Run("handles repository error", func(t *testing.T) {
		mockLoanRepo, _, _, job := newFunction(logger)
		mockLoanRepo.On("GetAllActiveLoanIDs", ctx).Return(nil, fmt.Errorf("%w: failed to query active loans: %w", apperrors.ErrDatabase, nil))

		err := job.Run(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		mockLoanRepo.AssertExpectations(t)
	})

	t.Run("handles loan service error", func(t *testing.T) {
		activeLoanIDs := []int64{1}
		mockLoanRepo, mockLoanService, _, job := newFunction(logger)
		mockLoanRepo.On("GetAllActiveLoanIDs", ctx).Return(activeLoanIDs, nil)

		mockLoanService.On("IsDelinquent", ctx, int64(1)).Return(false, errors.New("loan service error"))

		err := job.Run(ctx)
		assert.Error(t, err)

		mockLoanRepo.AssertExpectations(t)
		mockLoanService.AssertExpectations(t)
	})

	t.Run("handles customer service error", func(t *testing.T) {
		activeLoanIDs := []int64{1}
		mockLoanRepo, mockLoanService, mockCustomerService, job := newFunction(logger)
		mockLoanRepo.On("GetAllActiveLoanIDs", ctx).Return(activeLoanIDs, nil)

		mockLoanService.On("IsDelinquent", ctx, int64(1)).Return(true, nil)
		mockCustomerService.On("FindCustomerByLoan", ctx, int64(1)).Return(nil, errors.New("customer service error"))

		err := job.Run(ctx)
		assert.Error(t, err)

		mockLoanRepo.AssertExpectations(t)
		mockLoanService.AssertExpectations(t)
		mockCustomerService.AssertExpectations(t)
	})

	t.Run("handles no active loans", func(t *testing.T) {
		mockLoanRepo, _, _, job := newFunction(logger)
		mockLoanRepo.On("GetAllActiveLoanIDs", ctx).Return([]int64{}, nil)

		err := job.Run(ctx)
		assert.NoError(t, err)

		mockLoanRepo.AssertExpectations(t)
	})
}

func newFunction(logger *slog.Logger) (*MockLoanRepository, *MockLoanService, *MockCustomerService, *batch.UpdateDelinquencyJob) {
	mockLoanRepo := new(MockLoanRepository)
	mockLoanService := new(MockLoanService)
	mockCustomerService := new(MockCustomerService)

	job := batch.NewUpdateDelinquencyJob(mockLoanRepo, mockLoanService, mockCustomerService, logger)
	return mockLoanRepo, mockLoanService, mockCustomerService, job
}
