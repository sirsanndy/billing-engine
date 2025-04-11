package batch

import (
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/pkg/apperrors"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type UpdateDelinquencyJob struct {
	loanRepo        loan.Repository
	loanService     loan.LoanService
	customerService customer.CustomerService
	logger          *slog.Logger
}

func NewUpdateDelinquencyJob(
	loanRepo loan.Repository,
	loanSvc loan.LoanService,
	customerSvc customer.CustomerService,
	logger *slog.Logger,
) *UpdateDelinquencyJob {
	if loanRepo == nil || loanSvc == nil || customerSvc == nil || logger == nil {
		panic("UpdateDelinquencyJob dependencies cannot be nil")
	}
	return &UpdateDelinquencyJob{
		loanRepo:        loanRepo,
		loanService:     loanSvc,
		customerService: customerSvc,
		logger:          logger.With("job", "UpdateDelinquency"),
	}
}

func (j *UpdateDelinquencyJob) Run(ctx context.Context) error {
	startTime := time.Now()
	j.logger.InfoContext(ctx, "Starting daily customer delinquency update job.")

	j.logger.DebugContext(ctx, "Fetching active loan IDs from repository.")
	activeLoanIDs, err := j.loanRepo.GetAllActiveLoanIDs(ctx)
	if err != nil {
		j.logger.ErrorContext(ctx, "Failed to get active loan IDs, aborting job.", slog.Any("error", err))
		return fmt.Errorf("cannot run job, failed to get active loans: %w", err)
	}
	j.logger.InfoContext(ctx, "Fetched active loan IDs.", slog.Int("count", len(activeLoanIDs)))

	if len(activeLoanIDs) == 0 {
		j.logger.InfoContext(ctx, "No active loans found to process.")
		j.logger.InfoContext(ctx, "Customer delinquency update job finished.", slog.Duration("duration", time.Since(startTime)))
		return nil
	}

	var wg sync.WaitGroup
	var processedCount, delinquentCount, updatedToDelinquent, updatedToNotDelinquent, errorCount int32

	for _, loanID := range activeLoanIDs {
		wg.Add(1)
		go func(currentLoanID int64) {
			defer wg.Done()

			logCtx := j.logger.With(slog.Int64("loanID", currentLoanID))
			isDelinquent := false
			var checkErr error

			logCtx.DebugContext(ctx, "Checking loan delinquency status.")
			isDelinquent, checkErr = j.loanService.IsDelinquent(ctx, currentLoanID)
			if checkErr != nil {
				if errors.Is(checkErr, apperrors.ErrNotFound) {
					logCtx.WarnContext(ctx, "Loan not found during delinquency check (potentially deleted recently?)", slog.Any("error", checkErr))
				} else {
					logCtx.ErrorContext(ctx, "Failed to check loan delinquency", slog.Any("error", checkErr))
					errorCount++
				}
				return
			}

			if isDelinquent {
				delinquentCount++
			}

			logCtx.DebugContext(ctx, "Finding customer associated with loan.")
			cust, custErr := j.customerService.FindCustomerByLoan(ctx, currentLoanID)
			if custErr != nil {
				if errors.Is(custErr, customer.ErrNotFound) || errors.Is(custErr, apperrors.ErrNotFound) {
					logCtx.WarnContext(ctx, "No customer found linked to this loan (data inconsistency?)", slog.Any("error", custErr))
				} else {
					logCtx.ErrorContext(ctx, "Failed to find customer by loan", slog.Any("error", custErr))
					errorCount++
				}
				return
			}
			logCtx = logCtx.With(slog.Int64("customerID", cust.CustomerID))

			if cust.IsDelinquent != isDelinquent {
				logCtx.InfoContext(ctx, "Updating customer delinquency status.", slog.Bool("new_status", isDelinquent))
				updateErr := j.customerService.UpdateDelinquency(ctx, cust.CustomerID, isDelinquent)
				if updateErr != nil {
					logCtx.ErrorContext(ctx, "Failed to update customer delinquency status", slog.Any("error", updateErr))
					errorCount++
				} else {
					logCtx.InfoContext(ctx, "Customer delinquency status updated successfully.", slog.Bool("status", isDelinquent))
					if isDelinquent {
						updatedToDelinquent++
					} else {
						updatedToNotDelinquent++
					}
				}
			} else {
				logCtx.DebugContext(ctx, "Customer delinquency status already correct.", slog.Bool("status", isDelinquent))
			}
			processedCount++

		}(loanID)
	}

	wg.Wait()
	duration := time.Since(startTime)
	summaryLog := j.logger.With(
		slog.Duration("duration", duration),
		slog.Int("total_active_loans", len(activeLoanIDs)),
		slog.Int("loans_processed", int(processedCount)),
		slog.Int("loans_found_delinquent", int(delinquentCount)),
		slog.Int("customers_updated_to_delinquent", int(updatedToDelinquent)),
		slog.Int("customers_updated_to_not_delinquent", int(updatedToNotDelinquent)),
		slog.Int("errors_encountered", int(errorCount)),
	)
	if errorCount > 0 {
		summaryLog.WarnContext(ctx, "Customer delinquency update job finished with errors.")
	} else {
		summaryLog.InfoContext(ctx, "Customer delinquency update job finished successfully.")
	}

	if errorCount > 0 {
		return fmt.Errorf("job completed with %d errors", errorCount)
	}
	return nil
}
