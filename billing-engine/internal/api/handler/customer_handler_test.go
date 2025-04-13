package handler_test

import (
	"billing-engine/internal/api/handler"
	"billing-engine/internal/api/handler/dto"
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/pkg/apperrors"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func TestCreateCustomer(t *testing.T) {
	mockService := new(MockCustomerService)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := handler.NewCustomerHandler(mockService, logger)

	t.Run("success", func(t *testing.T) {
		reqBody := dto.CreateCustomerRequest{Name: "John Doe", Address: "123 Main St"}
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewReader(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mockCustomer := &customer.Customer{CustomerID: 1, Name: "John Doe", Address: "123 Main St"}
		mockService.On("CreateNewCustomer", mock.Anything, reqBody.Name, reqBody.Address).Return(mockCustomer, nil)

		handler.CreateCustomer(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp dto.CustomerResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(mockCustomer.CustomerID, 10), resp.CustomerID)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateCustomer(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		mockService.AssertNotCalled(t, "CreateNewCustomer")
	})
}

func TestGetCustomer(t *testing.T) {
	mockService := new(MockCustomerService)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := handler.NewCustomerHandler(mockService, logger)

	t.Run("success", func(t *testing.T) {
		mockCustomer := &customer.Customer{CustomerID: 1, Name: "John Doe", Address: "123 Main St"}
		mockService.On("GetCustomer", mock.Anything, int64(1)).Return(mockCustomer, nil)

		req := httptest.NewRequest(http.MethodGet, "/customers/1", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("customerID", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetCustomer(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp dto.CustomerResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(mockCustomer.CustomerID, 10), resp.CustomerID)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid customer ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/customers/abc", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("customerID", "abc")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetCustomer(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		mockService.AssertNotCalled(t, "GetCustomer")
	})

	t.Run("customer not found", func(t *testing.T) {
		mockService.On("GetCustomer", mock.Anything, int64(2)).Return(nil, apperrors.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/customers/2", nil)
		rec := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("customerID", "2")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.GetCustomer(rec, req)
		assert.NotNil(t, rec)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		mockService.AssertExpectations(t)
	})
}
