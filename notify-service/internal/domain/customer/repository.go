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
	Upsert(ctx context.Context, cust *Customer) error
}
