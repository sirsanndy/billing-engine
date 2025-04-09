package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("resource not found")

	ErrInvalidArgument = errors.New("invalid argument")

	ErrValidation = errors.New("validation failed")

	ErrAlreadyExists = errors.New("resource already exists")

	ErrDatabase = errors.New("database error")

	ErrInternalServer = errors.New("internal server error")

	ErrInvalidPaymentAmount = errors.New("invalid payment amount")

	ErrLoanFullyPaid = errors.New("loan is already fully paid")

	ErrUnauthorized = errors.New("unauthorized")

	ErrForbidden = errors.New("forbidden")

	ErrConflict = errors.New("resource conflict")
)

type ValidationError struct {
	Field   string
	Message string
	Cause   error
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}

func NewValidationError(field, message string) error {

	return fmt.Errorf("%w: %w", ErrValidation, &ValidationError{Field: field, Message: message})
}

type AppError struct {
	Code    string
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func WrapDatabaseError(cause error, message string) error {
	return &AppError{
		Code:    "DB_ERROR",
		Message: message,
		Cause:   fmt.Errorf("%w: %w", ErrDatabase, cause),
	}
}
