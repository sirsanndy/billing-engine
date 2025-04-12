package loan

import (
	"billing-engine/internal/domain/customer"
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

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

func TestCreateLoan(t *testing.T) {
	mockRepo := new(MockRepository)
	mockCustomerService := new(MockCustomerService)
	service := NewLoanService(mockRepo, mockCustomerService, logger)

	ctx := context.Background()
	principal := Money(1000)
	termWeeks := 52
	annualInterestRate := Money(5)
	startDate := time.Now()
	customerID := int64(1)

	loan := &Loan{}
	mockRepo.On("CreateLoan", ctx, mock.Anything, mock.Anything, mock.Anything).Return(loan, nil)
	mockCustomerService.On("GetCustomer", ctx, customerID).Return(&customer.Customer{CustomerID: customerID, Active: true}, nil)
	mockCustomerService.On("AssignLoanToCustomer", ctx, customerID, mock.Anything).Return(nil)

	result, err := service.CreateLoan(ctx, customerID, principal, termWeeks, annualInterestRate, startDate)

	assert.NoError(t, err)
	assert.Equal(t, loan, result)
	mockRepo.AssertExpectations(t)
}

func TestGetOutstanding(t *testing.T) {
	mockRepo := new(MockRepository)

	mockCustomerService := new(MockCustomerService)
	service := NewLoanService(mockRepo, mockCustomerService, logger)

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

	mockCustomerService := new(MockCustomerService)
	service := NewLoanService(mockRepo, mockCustomerService, logger)

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
	type TxMock struct {
		pgx.Tx
	}
	mockRepo := new(MockRepository)

	mockCustomerService := new(MockCustomerService)
	service := NewLoanService(mockRepo, mockCustomerService, logger)

	ctx := context.Background()
	loanID := int64(1)
	amount := Money(100)
	tx := &TxMock{}
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

	mockCustomerService := new(MockCustomerService)
	service := NewLoanService(mockRepo, mockCustomerService, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedLoan := &Loan{}

	mockRepo.On("GetLoanByID", ctx, loanID).Return(expectedLoan, nil)
	mockRepo.On("GetScheduleByLoanID", ctx, loanID).Return([]ScheduleEntry{}, nil)

	result, err := service.GetLoan(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedLoan, result)
	mockRepo.AssertExpectations(t)
}

func TestGetLoanSchedule(t *testing.T) {
	mockRepo := new(MockRepository)

	mockCustomerService := new(MockCustomerService)
	service := NewLoanService(mockRepo, mockCustomerService, logger)

	ctx := context.Background()
	loanID := int64(1)
	expectedSchedule := []ScheduleEntry{{}, {}}

	mockRepo.On("GetScheduleByLoanID", ctx, loanID).Return(expectedSchedule, nil)

	result, err := service.GetLoanSchedule(ctx, loanID)

	assert.NoError(t, err)
	assert.Equal(t, expectedSchedule, result)
	mockRepo.AssertExpectations(t)
}
