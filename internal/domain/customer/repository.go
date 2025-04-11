package customer

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("customer not found")

	ErrDuplicateLoanID = errors.New("loan ID already assigned to another customer")

	ErrUpdateConflict = errors.New("update conflict detected")

	ErrCannotDeactivateActiveLoan = errors.New("cannot deactivate customer with active loan")

	ErrCustomerAlreadyHasLoan = errors.New("customer already has an assigned active loan")
)

type CustomerRepository interface {
	Save(ctx context.Context, customer *Customer) error

	FindByID(ctx context.Context, customerID int64) (*Customer, error)

	FindByLoanID(ctx context.Context, loanID int64) (*Customer, error)

	FindAll(ctx context.Context, activeOnly bool) ([]*Customer, error)

	Delete(ctx context.Context, customerID int64) error

	SetDelinquencyStatus(ctx context.Context, customerID int64, isDelinquent bool) error

	SetActiveStatus(ctx context.Context, customerID int64, isActive bool) error
}
