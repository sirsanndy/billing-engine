package handler

import (
	"billing-engine/internal/api/handler/dto"
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/pkg/apperrors"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type CustomerHandler struct {
	service customer.CustomerService
	logger  *slog.Logger
}

func NewCustomerHandler(s customer.CustomerService, l *slog.Logger) *CustomerHandler {
	if s == nil {
		panic("customer service cannot be nil")
	}
	if l == nil {
		panic("logger cannot be nil")
	}
	return &CustomerHandler{
		service: s,
		logger:  l.With("component", "CustomerHandler"),
	}
}
func getCustomerIDFromURL(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "customerID")
	if idStr == "" {
		return 0, fmt.Errorf("%w: customerID not found in URL path", apperrors.ErrInvalidArgument)
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%w: invalid customerID format in URL path: %s", apperrors.ErrInvalidArgument, idStr)
	}
	return id, nil
}

// CreateCustomer handles POST /customers
// @Summary Create a new customer
// @Description Creates a new customer record with name and address.
// @Tags Customers
// @Accept json
// @Produce json
// @Param request body dto.CreateCustomerRequest true "Customer creation request"
// @Success 201 {object} dto.CustomerResponse "Customer successfully created"
// @Failure 400 {object} dto.ErrorResponse "Invalid request payload (e.g., empty name/address)"
// @Failure 500 {object} dto.ErrorResponse "Internal server error during creation"
// @Router /customers [post]
// @Security BearerAuth
func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {

	h.logger.DebugContext(r.Context(), "Received create customer request")

	var req dto.CreateCustomerRequest
	if err := decodeJSON(r, &req); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode request body", slog.Any("error", err))
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Address) == "" {
		h.logger.WarnContext(r.Context(), "Validation failed: Name or Address is empty")
		respondError(w, fmt.Errorf("%w: name and address cannot be empty", apperrors.ErrInvalidArgument))
		return
	}
	h.logger.DebugContext(r.Context(), "Request validation passed")

	h.logger.DebugContext(r.Context(), "Calling customer service CreateNewCustomer")
	createdCustomer, err := h.service.CreateNewCustomer(r.Context(), req.Name, req.Address)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Service failed to create customer", slog.Any("error", err))
		respondError(w, err)
		return
	}

	resp := dto.NewCustomerResponse(createdCustomer)
	h.logger.InfoContext(r.Context(), "Customer created successfully", slog.String("customerID", resp.CustomerID))
	respondJSON(w, http.StatusCreated, resp)
}

// GetCustomer handles GET /customers/{customerID}
// @Summary Retrieve customer details
// @Description Retrieves details for a specific customer by their ID.
// @Tags Customers
// @Produce json
// @Param customerID path int true "Customer ID" Minimum(1)
// @Success 200 {object} dto.CustomerResponse "Customer details retrieved"
// @Failure 400 {object} dto.ErrorResponse "Invalid customer ID format"
// @Failure 404 {object} dto.ErrorResponse "Customer not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers/{customerID} [get]
// @Security BearerAuth
func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {

	customerID, err := getCustomerIDFromURL(r)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to get customer ID from URL", slog.Any("error", err))
		respondError(w, err) // Pass logger to respondError
		return
	}

	h.logger.DebugContext(r.Context(), "Received get customer request")

	h.logger.DebugContext(r.Context(), "Calling customer service GetCustomer")
	domainCustomer, err := h.service.GetCustomer(r.Context(), customerID)
	if err != nil {

		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) && !errors.Is(err, apperrors.ErrNotFound) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to get customer", slog.Any("error", err))
		respondError(w, err)
		return
	}

	resp := dto.NewCustomerResponse(domainCustomer)
	h.logger.InfoContext(r.Context(), "Customer retrieved successfully")
	respondJSON(w, http.StatusOK, resp)
}

// ListCustomers handles GET /customers
// @Summary List customers
// @Description Retrieves a list of customers. Currently lists active customers by default.
// @Tags Customers
// @Produce json
// @Param active query bool false "Filter by active status (behaviour depends on service implementation)" Example(true)
// @Success 200 {array} dto.CustomerResponse "List of customers"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers [get]
// @Security BearerAuth
func (h *CustomerHandler) ListCustomers(w http.ResponseWriter, r *http.Request) {

	h.logger.DebugContext(r.Context(), "Received list customers request")

	h.logger.DebugContext(r.Context(), "Calling customer service ListActiveCustomers")
	customers, err := h.service.ListActiveCustomers(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Service failed to list active customers", slog.Any("error", err))
		respondError(w, err)
		return
	}

	resp := make([]dto.CustomerResponse, len(customers))
	for i, cust := range customers {
		resp[i] = dto.NewCustomerResponse(cust)
	}

	h.logger.InfoContext(r.Context(), "Customers listed successfully", slog.Int("count", len(resp)))
	respondJSON(w, http.StatusOK, resp)
}

// UpdateCustomerAddress handles PUT /customers/{customerID}/address
// @Summary Update customer address
// @Description Updates the address for a specific customer.
// @Tags Customers
// @Accept json
// @Produce json
// @Param customerID path int true "Customer ID" Minimum(1)
// @Param request body dto.UpdateCustomerAddressRequest true "New address payload"
// @Success 204 "Address successfully updated"
// @Failure 400 {object} dto.ErrorResponse "Invalid customer ID or request payload (e.g., empty address)"
// @Failure 404 {object} dto.ErrorResponse "Customer not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers/{customerID}/address [put]
// @Security BearerAuth
func (h *CustomerHandler) UpdateCustomerAddress(w http.ResponseWriter, r *http.Request) {

	customerID, err := getCustomerIDFromURL(r)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to get customer ID from URL", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.DebugContext(r.Context(), "Received update customer address request")

	var req dto.UpdateCustomerAddressRequest
	if err := decodeJSON(r, &req); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode request body", slog.Any("error", err))
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	if strings.TrimSpace(req.Address) == "" {
		h.logger.WarnContext(r.Context(), "Validation failed: Address is empty")
		respondError(w, fmt.Errorf("%w: address cannot be empty", apperrors.ErrInvalidArgument))
		return
	}
	h.logger.DebugContext(r.Context(), "Request validation passed")

	h.logger.DebugContext(r.Context(), "Calling customer service UpdateCustomerAddress")
	err = h.service.UpdateCustomerAddress(r.Context(), customerID, req.Address)
	if err != nil {
		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to update customer address", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "Customer address updated successfully")
	respondJSON(w, http.StatusNoContent, nil)
}

// AssignLoanToCustomer handles PUT /customers/{customerID}/loan
// @Summary Assign a loan to a customer
// @Description Associates a loan ID with a specific customer. Fails if the customer already has a different loan assigned or if the loan ID is already in use by another customer.
// @Tags Customers
// @Accept json
// @Produce json
// @Param customerID path int true "Customer ID" Minimum(1)
// @Param request body dto.AssignLoanRequest true "Loan ID payload (loanId must be positive)"
// @Success 204 "Loan successfully assigned"
// @Failure 400 {object} dto.ErrorResponse "Invalid customer ID or request payload (e.g., invalid loan ID)"
// @Failure 404 {object} dto.ErrorResponse "Customer not found"
// @Failure 409 {object} dto.ErrorResponse "Conflict (e.g., customer already has loan, loan ID already assigned)"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers/{customerID}/loan [put]
// @Security BearerAuth
func (h *CustomerHandler) AssignLoanToCustomer(w http.ResponseWriter, r *http.Request) {

	customerID, err := getCustomerIDFromURL(r)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to get customer ID from URL", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.DebugContext(r.Context(), "Received assign loan to customer request")

	var req dto.AssignLoanRequest
	if err := decodeJSON(r, &req); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode request body", slog.Any("error", err))
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	if req.LoanID <= 0 {
		h.logger.WarnContext(r.Context(), "Validation failed: LoanID is invalid")
		respondError(w, fmt.Errorf("%w: loanId must be positive", apperrors.ErrInvalidArgument))
		return
	}
	h.logger.DebugContext(r.Context(), "Request validation passed")

	h.logger.DebugContext(r.Context(), "Calling customer service AssignLoanToCustomer")
	err = h.service.AssignLoanToCustomer(r.Context(), customerID, req.LoanID)
	if err != nil {
		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) &&
			!errors.Is(err, customer.ErrDuplicateLoanID) &&
			!(strings.Contains(err.Error(), "already assigned loan")) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to assign loan to customer", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "Loan assigned to customer successfully")
	respondJSON(w, http.StatusNoContent, nil)
}

// UpdateDelinquency handles PUT /customers/{customerID}/delinquency
// @Summary Update customer delinquency status
// @Description Sets the delinquency status for a specific customer.
// @Tags Customers
// @Accept json
// @Produce json
// @Param customerID path int true "Customer ID" Minimum(1)
// @Param request body dto.UpdateDelinquencyRequest true "Delinquency status payload (`isDelinquent`: true/false)"
// @Success 204 "Delinquency status successfully updated"
// @Failure 400 {object} dto.ErrorResponse "Invalid customer ID or request payload"
// @Failure 404 {object} dto.ErrorResponse "Customer not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers/{customerID}/delinquency [put]
// @Security BearerAuth
func (h *CustomerHandler) UpdateDelinquency(w http.ResponseWriter, r *http.Request) {

	customerID, err := getCustomerIDFromURL(r)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to get customer ID from URL", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.DebugContext(r.Context(), "Received update delinquency request")

	var req dto.UpdateDelinquencyRequest
	if err := decodeJSON(r, &req); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode request body", slog.Any("error", err))
		respondError(w, fmt.Errorf("%w: %v", apperrors.ErrInvalidArgument, err))
		return
	}

	h.logger.DebugContext(r.Context(), "Request decoded successfully")

	h.logger.DebugContext(r.Context(), "Calling customer service UpdateDelinquency")
	err = h.service.UpdateDelinquency(r.Context(), customerID, req.IsDelinquent)
	if err != nil {
		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to update delinquency status", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "Customer delinquency status updated successfully")
	respondJSON(w, http.StatusNoContent, nil)
}

// DeactivateCustomer handles DELETE /customers/{customerID}
// @Summary Deactivate a customer
// @Description Marks a customer account as inactive. Fails if the customer has an associated loan that is not paid off.
// @Tags Customers
// @Produce json
// @Param customerID path int true "Customer ID" Minimum(1)
// @Success 204 "Customer successfully deactivated"
// @Failure 400 {object} dto.ErrorResponse "Invalid customer ID"
// @Failure 404 {object} dto.ErrorResponse "Customer not found"
// @Failure 409 {object} dto.ErrorResponse "Conflict (e.g., customer has an active loan)"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers/{customerID} [delete]
// @Security BearerAuth
func (h *CustomerHandler) DeactivateCustomer(w http.ResponseWriter, r *http.Request) {

	customerID, err := getCustomerIDFromURL(r)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to get customer ID from URL", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.DebugContext(r.Context(), "Received deactivate customer request")

	h.logger.DebugContext(r.Context(), "Calling customer service DeactivateCustomer")
	err = h.service.DeactivateCustomer(r.Context(), customerID)
	if err != nil {
		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) &&
			!errors.Is(err, customer.ErrCannotDeactivateActiveLoan) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to deactivate customer", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "Customer deactivated successfully")
	respondJSON(w, http.StatusNoContent, nil)
}

// ReactivateCustomer handles PUT /customers/{customerID}/reactivate
// @Summary Reactivate a customer
// @Description Marks a customer account as active.
// @Tags Customers
// @Produce json
// @Param customerID path int true "Customer ID" Minimum(1)
// @Success 204 "Customer successfully reactivated"
// @Failure 400 {object} dto.ErrorResponse "Invalid customer ID"
// @Failure 404 {object} dto.ErrorResponse "Customer not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers/{customerID}/reactivate [put]
// @Security BearerAuth
func (h *CustomerHandler) ReactivateCustomer(w http.ResponseWriter, r *http.Request) {

	customerID, err := getCustomerIDFromURL(r)
	if err != nil {
		h.logger.WarnContext(r.Context(), "Failed to get customer ID from URL", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.DebugContext(r.Context(), "Received reactivate customer request")

	h.logger.DebugContext(r.Context(), "Calling customer service ReactivateCustomer")
	err = h.service.ReactivateCustomer(r.Context(), customerID)
	if err != nil {
		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to reactivate customer", slog.Any("error", err))
		respondError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "Customer reactivated successfully")
	respondJSON(w, http.StatusNoContent, nil)
}

// FindCustomerByLoan handles GET /customers?loan_id={loanID}
// @Summary Find customer by loan ID
// @Description Retrieves the customer associated with a specific loan ID.
// @Tags Customers
// @Produce json
// @Param loan_id query int true "Loan ID to search for" Minimum(1)
// @Success 200 {object} dto.CustomerResponse "Customer details retrieved"
// @Failure 400 {object} dto.ErrorResponse "Invalid or missing loan_id query parameter"
// @Failure 404 {object} dto.ErrorResponse "Customer not found for the given loan ID"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /customers [get]
// @Security BearerAuth
func (h *CustomerHandler) FindCustomerByLoan(w http.ResponseWriter, r *http.Request) {

	h.logger.DebugContext(r.Context(), "Received find customer by loan request")

	loanIDStr := r.URL.Query().Get("loan_id")
	if loanIDStr == "" {
		h.logger.WarnContext(r.Context(), "Missing loan_id query parameter")
		respondError(w, fmt.Errorf("%w: missing required query parameter 'loan_id'", apperrors.ErrInvalidArgument))
		return
	}
	loanID, err := strconv.ParseInt(loanIDStr, 10, 64)
	if err != nil || loanID <= 0 {
		h.logger.WarnContext(r.Context(), "Invalid loan_id query parameter format", slog.String("loan_id_str", loanIDStr), slog.Any("error", err))
		respondError(w, fmt.Errorf("%w: invalid loan_id format: %s", apperrors.ErrInvalidArgument, loanIDStr))
		return
	}

	h.logger.DebugContext(r.Context(), "Query parameter validation passed")

	h.logger.DebugContext(r.Context(), "Calling customer service FindCustomerByLoan")
	domainCustomer, err := h.service.FindCustomerByLoan(r.Context(), loanID)
	if err != nil {
		level := slog.LevelWarn
		if !errors.Is(err, customer.ErrNotFound) && !errors.Is(err, apperrors.ErrNotFound) {
			level = slog.LevelError
		}
		h.logger.Log(r.Context(), level, "Service failed to find customer by loan", slog.Any("error", err))
		respondError(w, err)
		return
	}

	resp := dto.NewCustomerResponse(domainCustomer)
	h.logger.InfoContext(r.Context(), "Customer found successfully by loan ID", slog.String("customerID", resp.CustomerID))
	respondJSON(w, http.StatusOK, resp)
}
