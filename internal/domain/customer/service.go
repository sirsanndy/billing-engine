package customer

import (
	"billing-engine/internal/event"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	inputValidationPassed = "Input validation passed"
	customerNotFound      = "Customer not found by repository"
)

type CustomerService interface {
	CreateNewCustomer(ctx context.Context, name, address string) (*Customer, error)
	GetCustomer(ctx context.Context, customerID int64) (*Customer, error)
	ListActiveCustomers(ctx context.Context) ([]*Customer, error)
	UpdateCustomerAddress(ctx context.Context, customerID int64, newAddress string) error
	AssignLoanToCustomer(ctx context.Context, customerID int64, loanID int64) error
	UpdateDelinquency(ctx context.Context, customerID int64, isDelinquent bool) error
	DeactivateCustomer(ctx context.Context, customerID int64) error
	ReactivateCustomer(ctx context.Context, customerID int64) error
	FindCustomerByLoan(ctx context.Context, loanID int64) (*Customer, error)
}

var _ CustomerService = (*customerService)(nil)

type customerService struct {
	repo   CustomerRepository
	pub    *event.EventPublisher
	logger *slog.Logger
}

func NewCustomerService(repo CustomerRepository, eventPublisher *event.EventPublisher, logger *slog.Logger) CustomerService {
	if repo == nil {
		panic("customer repository cannot be nil")
	}

	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		logger.Warn("Warning: No logger provided to NewCustomerService, using default stderr handler")
	}

	if eventPublisher == nil || &eventPublisher == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
		logger.Warn("Warning: No event publisher provided to NewCustomerService, using default event publisher")
	}

	return &customerService{
		repo:   repo,
		pub:    eventPublisher,
		logger: logger.With(slog.String("component", "customerService")),
	}
}

func NewCustomerEventPayload(cust *Customer) event.CustomerEventPayload {
	if cust == nil {
		return event.CustomerEventPayload{}
	}
	return event.CustomerEventPayload{
		CustomerID:   cust.CustomerID,
		Name:         cust.Name,
		Address:      cust.Address,
		IsDelinquent: cust.IsDelinquent,
		Active:       cust.Active,
		LoanID:       cust.LoanID,
		CreateDate:   cust.CreateDate,
		UpdatedAt:    cust.UpdatedAt,
	}
}

func (s *customerService) PublishCustomerUpdateEvent(ctx context.Context, customer *Customer) {
	if customer == nil {
		s.logger.ErrorContext(ctx, "Attempted to publish update event for nil customer")
		return
	}
	event := event.CustomerUpdatedEvent{
		Timestamp: time.Now(),
		Payload:   NewCustomerEventPayload(customer),
	}
	s.logger.With(slog.Int64("customerID", customer.CustomerID))

	if err := s.pub.PublishCustomerUpdated(ctx, event); err != nil {
		s.logger.ErrorContext(ctx, "Failed to publish customer update event", slog.Any("error", err))
	} else {
		s.logger.InfoContext(ctx, "Successfully published customer update event")
	}
}

func (s *customerService) CreateNewCustomer(ctx context.Context, name, address string) (*Customer, error) {
	s.logger.InfoContext(ctx, "Attempting to create new customer")

	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if name == "" {
		s.logger.WarnContext(ctx, "Validation failed: name is empty")

		return nil, errors.New("customer name cannot be empty")
	}
	if address == "" {
		s.logger.WarnContext(ctx, "Validation failed: address is empty", slog.String("name", name))
		return nil, errors.New("customer address cannot be empty")
	}

	s.logger = s.logger.With(slog.String("validated_name", name), slog.String("validated_address", address))
	s.logger.InfoContext(ctx, inputValidationPassed)

	customer := &Customer{
		Name:         name,
		Address:      address,
		IsDelinquent: false,
		Active:       true,
		LoanID:       nil,
	}
	s.logger.InfoContext(ctx, "Customer domain object created")

	s.logger.InfoContext(ctx, "Calling repository Save")
	err := s.repo.Save(ctx, customer)
	if err != nil {
		s.logger.ErrorContext(ctx, "Repository failed to save new customer", slog.Any("error", err))

		return nil, fmt.Errorf("failed to save new customer: %w", err)
	}
	s.logger = s.logger.With(slog.Int64("customerID", customer.CustomerID))
	s.logger.InfoContext(ctx, "Successfully saved new customer, publishing creation event")
	createdEvent := event.CustomerCreatedEvent{
		Timestamp: time.Now(),
		Payload:   NewCustomerEventPayload(customer),
	}
	if pubErr := s.pub.PublishCustomerCreated(ctx, createdEvent); pubErr != nil {
		s.logger.ErrorContext(ctx, "Customer created, but FAILED to publish creation event", slog.Any("error", pubErr))
	} else {
		s.logger.InfoContext(ctx, "Successfully published customer creation event")
	}
	s.logger = s.logger.With(slog.Int64("customerID", customer.CustomerID))
	s.logger.InfoContext(ctx, "Successfully created new customer")
	return customer, nil
}

func (s *customerService) GetCustomer(ctx context.Context, customerID int64) (*Customer, error) {
	s.logger.InfoContext(ctx, "Attempting to get customer by ID")

	s.logger.InfoContext(ctx, "Calling repository FindByID")
	customer, err := s.repo.FindByID(ctx, customerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, customerNotFound)

			return nil, ErrNotFound
		}

		s.logger.ErrorContext(ctx, "Repository error finding customer", slog.Any("error", err))

		return nil, fmt.Errorf("failed to get customer %d: %w", customerID, err)
	}

	s.logger.InfoContext(ctx, "Successfully retrieved customer")
	return customer, nil
}

func (s *customerService) ListActiveCustomers(ctx context.Context) ([]*Customer, error) {

	s.logger.InfoContext(ctx, "Attempting to list all active customers")

	s.logger.InfoContext(ctx, "Calling repository FindAll", slog.Bool("activeOnly", true))
	customers, err := s.repo.FindAll(ctx, true)
	if err != nil {
		s.logger.ErrorContext(ctx, "Repository error listing active customers", slog.Any("error", err))
		return nil, fmt.Errorf("failed to list active customers: %w", err)
	}

	s.logger.InfoContext(ctx, "Successfully retrieved active customers", slog.Int("count", len(customers)))
	return customers, nil
}

func (s *customerService) UpdateCustomerAddress(ctx context.Context, customerID int64, newAddress string) error {

	s.logger.InfoContext(ctx, "Attempting to update customer address")

	newAddress = strings.TrimSpace(newAddress)
	if newAddress == "" {
		s.logger.WarnContext(ctx, "Validation failed: new address is empty")
		return errors.New("new address cannot be empty")
	}
	s.logger = s.logger.With(slog.String("new_address", newAddress))
	s.logger.InfoContext(ctx, inputValidationPassed)

	s.logger.InfoContext(ctx, "Calling repository FindByID to get current customer data")
	customer, err := s.repo.FindByID(ctx, customerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, "Customer not found by repository for update")
			return ErrNotFound
		}
		s.logger.ErrorContext(ctx, "Repository error finding customer for update", slog.Any("error", err))
		return fmt.Errorf("cannot find customer %d to update address: %w", customerID, err)
	}
	s.logger = s.logger.With(slog.String("current_address", customer.Address))

	if customer.Address == newAddress {
		s.logger.InfoContext(ctx, "No address change needed, skipping save")
		return nil
	}
	customer.Address = newAddress
	s.logger.InfoContext(ctx, "Address updated in memory structure, preparing to save")

	s.logger.InfoContext(ctx, "Calling repository Save to persist address change")
	err = s.repo.Save(ctx, customer)
	if err != nil {
		s.logger.ErrorContext(ctx, "Repository failed to save updated address", slog.Any("error", err))

		if errors.Is(err, ErrNotFound) {
			s.logger.ErrorContext(ctx, "Customer disappeared before save completed")
			return ErrNotFound
		}

		return fmt.Errorf("failed to save updated address for customer %d: %w", customerID, err)
	}

	s.logger.InfoContext(ctx, "Successfully updated customer address in repository, publishing update event.")
	s.PublishCustomerUpdateEvent(ctx, customer)

	s.logger.InfoContext(ctx, "Successfully updated customer address")
	return nil
}

func (s *customerService) AssignLoanToCustomer(ctx context.Context, customerID int64, loanID int64) error {

	s.logger.InfoContext(ctx, "Attempting to assign loan to customer")

	if loanID <= 0 {
		s.logger.WarnContext(ctx, "Validation failed: invalid loan ID provided")
		return errors.New("invalid loan ID provided")
	}
	s.logger.InfoContext(ctx, inputValidationPassed)

	s.logger.InfoContext(ctx, "Calling repository FindByID")
	customer, err := s.repo.FindByID(ctx, customerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, customerNotFound)
			return ErrNotFound
		}
		s.logger.ErrorContext(ctx, "Repository error finding customer", slog.Any("error", err))
		return fmt.Errorf("cannot find customer %d to assign loan: %w", customerID, err)
	}
	s.logger = s.logger.With(slog.Bool("customer_active", customer.Active))

	if !customer.Active {
		s.logger.WarnContext(ctx, "Business rule failed: cannot assign loan to inactive customer")

		return fmt.Errorf("cannot assign loan to inactive customer %d", customerID)
	}

	if customer.LoanID != nil {
		s.logger = s.logger.With(slog.Int64("existing_loanID", *customer.LoanID))
		if *customer.LoanID == loanID {
			s.logger.InfoContext(ctx, "Loan already assigned to this customer, no action needed")
			return nil
		}
		s.logger.WarnContext(ctx, "Business rule failed: customer already has a different loan assigned")

		return fmt.Errorf("customer %d already assigned loan %d, cannot assign new loan %d", customerID, *customer.LoanID, loanID)
	}
	s.logger.InfoContext(ctx, "Business rules passed")

	customer.LoanID = &loanID
	s.logger.InfoContext(ctx, "Calling repository Save to persist loan assignment")
	err = s.repo.Save(ctx, customer)
	if err != nil {
		s.logger.ErrorContext(ctx, "Repository failed to save loan assignment", slog.Any("error", err))

		if errors.Is(err, ErrDuplicateLoanID) {
			s.logger.WarnContext(ctx, "Duplicate Loan ID conflict detected during save")

			return ErrDuplicateLoanID
		}
		if errors.Is(err, ErrNotFound) {
			s.logger.ErrorContext(ctx, "Customer disappeared before save could complete")
			return ErrNotFound
		}

		return fmt.Errorf("failed to save loan assignment for customer %d: %w", customerID, err)
	}

	s.logger.InfoContext(ctx, "Successfully assign loan to customer in repository, publishing update event.")
	s.PublishCustomerUpdateEvent(ctx, customer)

	s.logger.InfoContext(ctx, "Successfully assigned loan to customer")
	return nil
}

func (s *customerService) UpdateDelinquency(ctx context.Context, customerID int64, isDelinquent bool) error {
	s.logger.InfoContext(ctx, "Attempting to update customer delinquency status")

	s.logger.InfoContext(ctx, "Calling repository SetDelinquencyStatus")
	err := s.repo.SetDelinquencyStatus(ctx, customerID, isDelinquent)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, customerNotFound)
			return ErrNotFound
		}

		s.logger.ErrorContext(ctx, "Repository error updating delinquency status", slog.Any("error", err))
		return fmt.Errorf("failed to update delinquency for customer %d: %w", customerID, err)
	}
	updatedCustomer, fetchErr := s.repo.FindByID(ctx, customerID)
	s.logger.InfoContext(ctx, "Successfully update delinquency status to customer in repository, publishing update event.")
	if fetchErr != nil {
		s.logger.ErrorContext(ctx, "Successfully updated status, but FAILED to re-fetch customer for event publishing", slog.Any("error", fetchErr))
	} else {
		s.PublishCustomerUpdateEvent(ctx, updatedCustomer)
	}
	s.logger.InfoContext(ctx, "Successfully updated customer delinquency status")
	return nil
}

func (s *customerService) DeactivateCustomer(ctx context.Context, customerID int64) error {

	s.logger.InfoContext(ctx, "Attempting to deactivate customer")

	s.logger.InfoContext(ctx, "Calling repository SetActiveStatus", slog.Bool("isActive", false))
	err := s.repo.SetActiveStatus(ctx, customerID, false)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, customerNotFound)
			return ErrNotFound
		}
		s.logger.ErrorContext(ctx, "Repository error deactivating customer", slog.Any("error", err))
		return fmt.Errorf("failed to deactivate customer %d: %w", customerID, err)
	}

	s.logger.InfoContext(ctx, "Successfully deactivate customer in repository, publishing update event.")
	deactivateCustomer, fetchErr := s.repo.FindByID(ctx, customerID)
	if fetchErr != nil {
		s.logger.ErrorContext(ctx, "Successfully updated status, but FAILED to re-fetch customer for event publishing", slog.Any("error", fetchErr))
	} else {
		s.PublishCustomerUpdateEvent(ctx, deactivateCustomer)
	}
	s.logger.InfoContext(ctx, "Successfully deactivated customer")
	return nil
}

func (s *customerService) ReactivateCustomer(ctx context.Context, customerID int64) error {

	s.logger.InfoContext(ctx, "Attempting to reactivate customer")

	s.logger.InfoContext(ctx, "Calling repository SetActiveStatus", slog.Bool("isActive", true))
	err := s.repo.SetActiveStatus(ctx, customerID, true)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, customerNotFound)
			return ErrNotFound
		}
		s.logger.ErrorContext(ctx, "Repository error reactivating customer", slog.Any("error", err))
		return fmt.Errorf("failed to reactivate customer %d: %w", customerID, err)
	}

	s.logger.InfoContext(ctx, "Successfully reactivate customer in repository, publishing update event.")
	reactivateCustomer, fetchErr := s.repo.FindByID(ctx, customerID)
	if fetchErr != nil {
		s.logger.ErrorContext(ctx, "Successfully updated status, but FAILED to re-fetch customer for event publishing", slog.Any("error", fetchErr))
	} else {
		s.PublishCustomerUpdateEvent(ctx, reactivateCustomer)
	}

	s.logger.InfoContext(ctx, "Successfully reactivated customer")
	return nil
}

func (s *customerService) FindCustomerByLoan(ctx context.Context, loanID int64) (*Customer, error) {

	s.logger.InfoContext(ctx, "Attempting to find customer by loan ID")

	s.logger.InfoContext(ctx, "Calling repository FindByLoanID")
	customer, err := s.repo.FindByLoanID(ctx, loanID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.WarnContext(ctx, "Customer not found by repository for this loan ID")
			return nil, ErrNotFound
		}
		s.logger.ErrorContext(ctx, "Repository error finding customer by loan ID", slog.Any("error", err))
		return nil, fmt.Errorf("failed to find customer by loan ID %d: %w", loanID, err)
	}

	s.logger = s.logger.With(slog.Int64("found_customerID", customer.CustomerID))
	s.logger.InfoContext(ctx, "Successfully found customer by loan ID")
	return customer, nil
}
