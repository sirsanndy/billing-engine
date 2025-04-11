package customer

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockCustomerRepository struct {
	mock.Mock
}

func (_m *MockCustomerRepository) Save(ctx context.Context, customer *Customer) error {
	ret := _m.Called(ctx, customer)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *Customer) error); ok {
		r0 = rf(ctx, customer)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerRepository) FindByID(ctx context.Context, customerID int64) (*Customer, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *Customer
	if rf, ok := ret.Get(0).(func(context.Context, int64) *Customer); ok {
		r0 = rf(ctx, customerID)
	} else {

		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Customer)
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

func (_m *MockCustomerRepository) FindByLoanID(ctx context.Context, loanID int64) (*Customer, error) {
	ret := _m.Called(ctx, loanID)

	var r0 *Customer
	if rf, ok := ret.Get(0).(func(context.Context, int64) *Customer); ok {
		r0 = rf(ctx, loanID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Customer)
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

func (_m *MockCustomerRepository) FindAll(ctx context.Context, activeOnly bool) ([]*Customer, error) {
	ret := _m.Called(ctx, activeOnly)

	var r0 []*Customer
	if rf, ok := ret.Get(0).(func(context.Context, bool) []*Customer); ok {
		r0 = rf(ctx, activeOnly)
	} else {

		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*Customer)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, bool) error); ok {
		r1 = rf(ctx, activeOnly)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockCustomerRepository) Delete(ctx context.Context, customerID int64) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerRepository) SetDelinquencyStatus(ctx context.Context, customerID int64, isDelinquent bool) error {
	ret := _m.Called(ctx, customerID, isDelinquent)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, bool) error); ok {
		r0 = rf(ctx, customerID, isDelinquent)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *MockCustomerRepository) SetActiveStatus(ctx context.Context, customerID int64, isActive bool) error {
	ret := _m.Called(ctx, customerID, isActive)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, bool) error); ok {
		r0 = rf(ctx, customerID, isActive)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

var _ CustomerRepository = (*MockCustomerRepository)(nil)
