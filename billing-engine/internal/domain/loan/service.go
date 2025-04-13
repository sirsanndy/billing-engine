package loan

import (
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/infrastructure/monitoring"
	"billing-engine/internal/pkg/apperrors"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

type Money = float64

type LoanService interface {
	CreateLoan(ctx context.Context, customerID int64, principal Money, termWeeks int, annualInterestRate Money, startDate time.Time) (*Loan, error)

	GetOutstanding(ctx context.Context, loanID int64) (Money, error)

	IsDelinquent(ctx context.Context, loanID int64) (bool, error)

	MakePayment(ctx context.Context, loanID int64, amount Money) error

	GetLoan(ctx context.Context, loanID int64) (*Loan, error)

	GetLoanSchedule(ctx context.Context, loanID int64) ([]ScheduleEntry, error)
}

type loanServiceImpl struct {
	repo            Repository
	customerService customer.CustomerService
	logger          *slog.Logger
}

func NewLoanService(r Repository, cs customer.CustomerService, logger *slog.Logger) LoanService {
	return &loanServiceImpl{repo: r, customerService: cs, logger: logger}
}

func (s *loanServiceImpl) CreateLoan(ctx context.Context, customerID int64, principal Money, termWeeks int, annualInterestRate Money, startDate time.Time) (*Loan, error) {
	s.logger.Info("Creating new loan")
	cust, err := s.customerService.GetCustomer(ctx, customerID)
	if err != nil {
		if errors.Is(err, customer.ErrNotFound) || errors.Is(err, apperrors.ErrNotFound) {
			s.logger.Error("Customer not found", slog.Any("error", err))
			return nil, fmt.Errorf("%w: customer %d not found", apperrors.ErrValidation, customerID)
		}
		s.logger.Error("Failed to get customer details from customer service", slog.Any("error", err))
		return nil, fmt.Errorf("failed to verify customer status: %w", err)
	}

	if !cust.Active {
		s.logger.Error("Attempted to create loan for inactive customer")
		return nil, fmt.Errorf("%w: customer %d is not active", apperrors.ErrValidation, customerID)
	}

	if cust.LoanID != nil {
		existingLoanID := *cust.LoanID
		existingLoan, err := s.GetLoan(ctx, existingLoanID)
		if err != nil {
			s.logger.Error("Failed to get existing loan details", "error", err)
			return nil, fmt.Errorf("failed to get existing loan details: %w", err)
		}

		if existingLoan.Status != StatusPaidOff {
			s.logger.Error("Customer already has an assigned active loan")
			return nil, fmt.Errorf("%w (LoanID: %d)", customer.ErrCustomerAlreadyHasLoan, existingLoanID)
		}
	}

	loan, err := NewLoan(principal, termWeeks, annualInterestRate, startDate)
	if err != nil {
		s.logger.Error("Failed to create new loan object", "error", err)
		return nil, fmt.Errorf("failed to create new loan object: %w", err)
	}

	schedule, err := loan.GenerateSchedule()
	if err != nil {
		s.logger.Error("Failed to generate loan schedule", "error", err)
		return nil, fmt.Errorf("failed to generate schedule: %w", err)
	}

	createdLoan, err := s.repo.CreateLoan(ctx, customerID, loan, schedule)
	if err != nil {
		s.logger.Error("Failed to save loan and schedule", "error", err)
		return nil, fmt.Errorf("%w: failed to save loan and schedule: %v", apperrors.ErrInternalServer, err)
	}

	s.customerService.AssignLoanToCustomer(ctx, customerID, createdLoan.ID)
	if err != nil {
		s.logger.Error("Failed to assign loan to customer", "error", err)
		return nil, fmt.Errorf("failed to assign loan to customer: %w", err)
	}
	s.logger.Info("Loan created successfully", "loanID", createdLoan.ID, "customerID", customerID)

	return createdLoan, nil
}

func (s *loanServiceImpl) GetOutstanding(ctx context.Context, loanID int64) (Money, error) {
	s.logger.Info("Getting total outstanding amount for loan", "loanID", loanID)
	outstandingAmount, err := s.repo.GetTotalOutstandingAmount(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("Loan not found", "loanID", loanID)
			return 0, fmt.Errorf("%w: loan with ID %d not found", apperrors.ErrNotFound, loanID)
		}
		s.logger.Warn("Failed to get outstanding amount", "loanID", loanID, "error", err)
		return 0, fmt.Errorf("%w: failed to get outstanding amount for loan %d: %v", apperrors.ErrInternalServer, loanID, err)
	}

	return outstandingAmount, nil
}

func (s *loanServiceImpl) IsDelinquent(ctx context.Context, loanID int64) (bool, error) {
	s.logger.Info("Checking if loan is delinquent", "loanID", loanID)
	lastTwoUnpaid, err := s.repo.GetLastTwoDueUnpaidSchedules(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("Loan not found", "loanID", loanID)
			return false, fmt.Errorf("%w: loan with ID %d not found for delinquency check", apperrors.ErrNotFound, loanID)
		}
		s.logger.Warn("Failed to check delinquency", "loanID", loanID, "error", err)
		return false, fmt.Errorf("%w: failed to check delinquency for loan %d: %v", apperrors.ErrInternalServer, loanID, err)
	}

	return len(lastTwoUnpaid) >= 2, nil
}

func (s *loanServiceImpl) MakePayment(ctx context.Context, loanID int64, amount Money) (err error) {
	s.logger.Info("Making payment", "loanID", loanID, "amount", amount)
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("%w: could not begin transaction: %v", apperrors.ErrInternalServer, err)
	}

	defer func() {
		status := "failure_internal"
		if errors.Is(err, apperrors.ErrInvalidPaymentAmount) {
			s.logger.Error("Invalid payment amount", "loanID", loanID, "amount", amount, "error", err)
			status = "failure_amount"
		}
		if errors.Is(err, apperrors.ErrLoanFullyPaid) {
			s.logger.Error("Loan is already fully paid", "loanID", loanID, "error", err)
			status = "failure_fully_paid"
		}
		monitoring.RecordPayment(status)
		if p := recover(); p != nil {
			s.logger.Error("Panic occurred during payment processing", "loanID", loanID, "error", p)
			_ = s.repo.RollbackTx(ctx, tx)
			panic(p)
		} else if err != nil {
			s.logger.Error("Rolling back transaction due to error :", "error", err)
			_ = s.repo.RollbackTx(ctx, tx)
		}

	}()

	entry, err := s.repo.FindOldestUnpaidEntryForUpdate(ctx, tx, loanID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			s.logger.Error("Loan is already fully paid", "loanID", loanID, "error", err)
			return apperrors.ErrLoanFullyPaid
		}

		if errors.Is(err, pgx.ErrNoRows) {
			_, checkLoanErr := s.repo.GetLoanByID(ctx, loanID)
			if errors.Is(checkLoanErr, pgx.ErrNoRows) || errors.Is(checkLoanErr, apperrors.ErrNotFound) {
				s.logger.Error("Loan not found", "loanID", loanID, "error", checkLoanErr)
				return fmt.Errorf("%w: cannot make payment, loan ID %d not found", apperrors.ErrNotFound, loanID)
			}

			return apperrors.ErrLoanFullyPaid
		}
		s.logger.Error("Failed to find schedule entry to pay", "loanID", loanID, "error", err)
		return fmt.Errorf("%w: could not find schedule entry to pay: %v", apperrors.ErrInternalServer, err)
	}

	tolerance := 0.001
	if math.Abs(amount-entry.DueAmount) > tolerance {
		s.logger.Error("Payment amount does not match due amount", "loanID", loanID, "amount", amount, "dueAmount", entry.DueAmount)
		return fmt.Errorf("%w: payment amount %.2f does not match due amount %.2f",
			apperrors.ErrInvalidPaymentAmount, amount, entry.DueAmount)
	}

	now := time.Now()
	entry.Status = PaymentStatusPaid
	entry.PaidAmount = amount
	entry.PaymentDate = &now
	entry.UpdatedAt = now

	err = s.repo.UpdateScheduleEntryInTx(ctx, tx, entry)
	if err != nil {
		s.logger.Error("Failed to update schedule entry", "loanID", loanID, "error", err)
		return fmt.Errorf("%w: could not update schedule entry: %v", apperrors.ErrInternalServer, err)
	}

	allPaid, err := s.repo.CheckIfAllPaymentsMadeInTx(ctx, tx, loanID)
	if err != nil {
		s.logger.Error("Failed to check if all payments are made", "loanID", loanID, "error", err)
		return fmt.Errorf("%w: could not check if loan payments are complete: %v", apperrors.ErrInternalServer, err)
	}

	if allPaid {
		err = s.repo.UpdateLoanStatusInTx(ctx, tx, loanID, StatusPaidOff)
		if err != nil {
			s.logger.Error("Failed to update loan status to paid off", "loanID", loanID, "error", err)
			return fmt.Errorf("%w: could not update loan status to paid off: %v", apperrors.ErrInternalServer, err)
		}
	}

	err = s.repo.CommitTx(ctx, tx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", "loanID", loanID, "error", err)
		return fmt.Errorf("%w: could not commit transaction: %v", apperrors.ErrInternalServer, err)
	}
	monitoring.RecordPayment("success")
	s.logger.Info("Payment processed successfully", "loanID", loanID, "amount", amount)
	return nil
}

func (s *loanServiceImpl) GetLoan(ctx context.Context, loanID int64) (*Loan, error) {
	s.logger.Info("Getting loan details", "loanID", loanID)
	loan, err := s.repo.GetLoanByID(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("Loan not found", "loanID", loanID)
			return nil, fmt.Errorf("%w: loan with ID %d not found", apperrors.ErrNotFound, loanID)
		}

		s.logger.Error("Failed to get loan", "loanID", loanID, "error", err)
		return nil, fmt.Errorf("%w: failed to get loan %d: %v", apperrors.ErrInternalServer, loanID, err)
	}

	schedule, err := s.GetLoanSchedule(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("Loan not found", "loanID", loanID)
		} else {
			s.logger.Error("Failed to get loan schedule", "loanID", loanID, "error", err)
		}
	}

	loan.Schedule = schedule
	return loan, nil
}

func (s *loanServiceImpl) GetLoanSchedule(ctx context.Context, loanID int64) ([]ScheduleEntry, error) {
	s.logger.Info("Getting loan schedule", "loanID", loanID)
	schedule, err := s.repo.GetScheduleByLoanID(ctx, loanID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.logger.Warn("Loan not found", "loanID", loanID)
			return nil, fmt.Errorf("%w: loan with ID %d not found when getting schedule", apperrors.ErrNotFound, loanID)
		}
		return nil, fmt.Errorf("%w: failed to get schedule for loan %d: %v", apperrors.ErrInternalServer, loanID, err)
	}
	if len(schedule) == 0 {
		_, checkLoanErr := s.repo.GetLoanByID(ctx, loanID)
		if errors.Is(checkLoanErr, pgx.ErrNoRows) || errors.Is(checkLoanErr, apperrors.ErrNotFound) {
			s.logger.Warn("Loan not found", "loanID", loanID)
			return nil, fmt.Errorf("%w: loan with ID %d not found when getting schedule", apperrors.ErrNotFound, loanID)
		}

	}
	return schedule, nil
}
