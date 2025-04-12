package handler

import (
	"billing-engine/internal/api/handler/dto"
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/pkg/apperrors"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
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

func TestLoanHandlerGetLoan(t *testing.T) {
	mockService := new(MockLoanService)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	handler := NewLoanHandler(mockService, logger)

	t.Run("successfully retrieves loan details", func(t *testing.T) {
		loanID := int64(123)
		mockLoan := &loan.Loan{
			ID: loanID,
		}

		mockService.On("GetLoan", mock.Anything, loanID).Return(mockLoan, nil)

		req := httptest.NewRequest(http.MethodGet, "/loans/123", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
			URLParams: chi.RouteParams{Keys: []string{"loanID"}, Values: []string{"123"}},
		}))
		rec := httptest.NewRecorder()

		handler.GetLoan(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp dto.LoanResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "123", resp.ID)
		mockService.AssertExpectations(t)
	})

	t.Run("returns error for invalid loan ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/loans/invalid", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
			URLParams: chi.RouteParams{Keys: []string{"loanID"}, Values: []string{"invalid"}},
		}))
		rec := httptest.NewRecorder()

		handler.GetLoan(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var resp dto.ErrorResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error.Message, "invalid syntax")
	})

	t.Run("returns error when loan not found", func(t *testing.T) {
		loanID := int64(2)
		mockService.On("GetLoan", mock.Anything, loanID).Return((*loan.Loan)(nil), apperrors.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/loans/2", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
			URLParams: chi.RouteParams{Keys: []string{"loanID"}, Values: []string{"2"}},
		}))
		rec := httptest.NewRecorder()

		handler.GetLoan(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		var resp dto.ErrorResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "Resource not found.", resp.Error.Message)
		mockService.AssertExpectations(t)
	})

	t.Run("returns internal server error for unexpected errors", func(t *testing.T) {
		loanID := int64(3)
		mockService.On("GetLoan", mock.Anything, loanID).Return((*loan.Loan)(nil), errors.New("unexpected error"))

		req := httptest.NewRequest(http.MethodGet, "/loans/3", nil)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
			URLParams: chi.RouteParams{Keys: []string{"loanID"}, Values: []string{"3"}},
		}))
		rec := httptest.NewRecorder()

		handler.GetLoan(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		var resp dto.ErrorResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, "An unexpected error occurred.", resp.Error.Message)
		mockService.AssertExpectations(t)
	})
}
