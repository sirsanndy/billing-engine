package handler

import (
	"billing-engine/internal/api/handler/dto"
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/pkg/apperrors"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type LoanHandler struct {
	service loan.LoanService
	logger  *slog.Logger
}

func NewLoanHandler(s loan.LoanService, l *slog.Logger) *LoanHandler {
	return &LoanHandler{
		service: s,
		logger:  l.With("component", "LoanHandler"),
	}
}

func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("no request body")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		slog.Default().Error("Failed to marshal JSON response", "error", err)
		http.Error(w, `{"error":{"message":"Internal server error"}}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}

func respondError(w http.ResponseWriter, err error) {
	status, message, field := http.StatusInternalServerError, "An unexpected error occurred.", ""
	var validationError *apperrors.ValidationError
	var appErr *apperrors.AppError

	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		status, message = http.StatusNotFound, "Resource not found."
	case errors.Is(err, apperrors.ErrInvalidArgument), errors.Is(err, apperrors.ErrValidation):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, apperrors.ErrInvalidPaymentAmount), errors.Is(err, apperrors.ErrLoanFullyPaid):
		status, message = http.StatusBadRequest, err.Error()
	case errors.As(err, &validationError):
		status, message, field = http.StatusBadRequest, validationError.Message, validationError.Field
	case errors.As(err, &appErr):
		message = appErr.Error()
	default:
		slog.Default().Error("Unhandled internal error", "error", err)
	}

	resp := dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Message: message,
			Field:   field,
		},
	}
	respondJSON(w, status, resp)
}

func getLoanIDFromURL(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "loanID")
	if idStr == "" {
		return 0, fmt.Errorf("loanID not found in URL path")
	}
	return strconv.ParseInt(idStr, 10, 64)
}

// CreateLoan handles the creation of a new loan.
//
// @Summary Create a new loan
// @Description This endpoint allows the creation of a new loan by providing the principal amount, term in weeks, annual interest rate, and start date.
// @Tags Loans
// @Accept json
// @Produce json
// @Param request body dto.CreateLoanRequest true "Loan creation request payload"
// @Success 201 {object} dto.LoanResponse "Loan successfully created"
// @Failure 400 {object} dto.ErrorResponse "Invalid request payload or validation error"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /loans [post]
// @Security BearerAuth
func (h *LoanHandler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateLoanRequest
	if err := decodeJSON(r, &req); err != nil || req.Validate() != nil {
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	startDate, _ := time.Parse(time.RFC3339[:10], req.StartDate)

	createdLoan, err := h.service.CreateLoan(r.Context(), req.CustomerID, req.Principal, req.TermWeeks, req.AnnualInterestRate, startDate)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := dto.NewLoanResponse(createdLoan, false)
	respondJSON(w, http.StatusCreated, resp)
}

// GetLoan retrieves the details of a specific loan.
//
// @Summary Retrieve loan details
// @Description This endpoint retrieves the details of a loan by its ID. Optionally, the repayment schedule can be included in the response by adding the query parameter `include=schedule`.
// @Tags Loans
// @Produce json
// @Param loanID path int true "Loan ID"
// @Param include query string false "Optional parameter to include repayment schedule (use 'schedule')"
// @Success 200 {object} dto.LoanResponse "Loan details successfully retrieved"
// @Failure 400 {object} dto.ErrorResponse "Invalid loan ID or request parameters"
// @Failure 404 {object} dto.ErrorResponse "Loan not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /loans/{loanID} [get]
// @Security BearerAuth
func (h *LoanHandler) GetLoan(w http.ResponseWriter, r *http.Request) {
	loanID, err := getLoanIDFromURL(r)
	if err != nil {
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	domainLoan, err := h.service.GetLoan(r.Context(), loanID)
	if err != nil {
		respondError(w, err)
		return
	}

	includeSchedule := r.URL.Query().Get("include") == "schedule"
	resp := dto.NewLoanResponse(domainLoan, includeSchedule)
	respondJSON(w, http.StatusOK, resp)
}

// GetOutstanding retrieves the outstanding amount for a specific loan.
//
// @Summary Retrieve outstanding loan amount
// @Description This endpoint retrieves the outstanding amount for a loan by its ID.
// @Tags Loans
// @Produce json
// @Param loanID path int true "Loan ID"
// @Success 200 {object} dto.OutstandingResponse "Outstanding amount successfully retrieved"
// @Failure 400 {object} dto.ErrorResponse "Invalid loan ID or request parameters"
// @Failure 404 {object} dto.ErrorResponse "Loan not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /loans/{loanID}/outstanding [get]
// @Security BearerAuth
func (h *LoanHandler) GetOutstanding(w http.ResponseWriter, r *http.Request) {
	loanID, err := getLoanIDFromURL(r)
	if err != nil {
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	outstandingAmountFloat, err := h.service.GetOutstanding(r.Context(), loanID)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := dto.OutstandingResponse{
		LoanID:            strconv.FormatInt(loanID, 10),
		OutstandingAmount: fmt.Sprintf("%.2f", outstandingAmountFloat),
	}
	respondJSON(w, http.StatusOK, resp)
}

// IsDelinquent checks if a loan is delinquent.
//
// @Summary Check loan delinquency status
// @Description This endpoint checks whether a loan is delinquent by its ID.
// @Tags Loans
// @Produce json
// @Param loanID path int true "Loan ID"
// @Success 200 {object} dto.DelinquentResponse "Delinquency status successfully retrieved"
// @Failure 400 {object} dto.ErrorResponse "Invalid loan ID or request parameters"
// @Failure 404 {object} dto.ErrorResponse "Loan not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /loans/{loanID}/delinquent [get]
// @Security BearerAuth
func (h *LoanHandler) IsDelinquent(w http.ResponseWriter, r *http.Request) {
	loanID, err := getLoanIDFromURL(r)
	if err != nil {
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	isDelinquent, err := h.service.IsDelinquent(r.Context(), loanID)
	if err != nil {
		respondError(w, err)
		return
	}

	resp := dto.DelinquentResponse{
		LoanID:       strconv.FormatInt(loanID, 10),
		IsDelinquent: isDelinquent,
	}
	respondJSON(w, http.StatusOK, resp)
}

// MakePayment processes a payment for a specific loan.
//
// @Summary Make a loan payment
// @Description This endpoint processes a payment for a loan by its ID. The payment amount must be specified in the request payload.
// @Tags Loans
// @Accept json
// @Produce json
// @Param loanID path int true "Loan ID"
// @Param request body dto.MakePaymentRequest true "Payment request payload"
// @Success 200 {object} map[string]string "Payment successfully processed"
// @Failure 400 {object} dto.ErrorResponse "Invalid loan ID, request payload, or validation error"
// @Failure 404 {object} dto.ErrorResponse "Loan not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /loans/{loanID}/payments [post]
// @Security BearerAuth
func (h *LoanHandler) MakePayment(w http.ResponseWriter, r *http.Request) {
	loanID, err := getLoanIDFromURL(r)
	if err != nil {
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	var req dto.MakePaymentRequest
	if err := decodeJSON(r, &req); err != nil || req.Validate() != nil {
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	amountDecimal, err := decimal.NewFromString(req.Amount)
	if err != nil {
		respondError(w, fmt.Errorf("%w: invalid numeric format for amount", apperrors.ErrInvalidArgument))
		return
	}
	amountFloat, _ := amountDecimal.Float64()

	if err := h.service.MakePayment(r.Context(), loanID, amountFloat); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Payment successful"})
}
