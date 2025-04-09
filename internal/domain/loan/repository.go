package loan

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	CreateLoan(ctx context.Context, loan *Loan, schedule []ScheduleEntry) (createdLoan *Loan, err error)

	GetLoanByID(ctx context.Context, loanID int64) (*Loan, error)

	GetScheduleByLoanID(ctx context.Context, loanID int64) ([]ScheduleEntry, error)

	GetUnpaidSchedules(ctx context.Context, loanID int64) ([]ScheduleEntry, error)

	GetLastTwoDueUnpaidSchedules(ctx context.Context, loanID int64) ([]ScheduleEntry, error)

	FindOldestUnpaidEntryForUpdate(ctx context.Context, tx pgx.Tx, loanID int64) (*ScheduleEntry, error)

	UpdateScheduleEntryInTx(ctx context.Context, tx pgx.Tx, entry *ScheduleEntry) error

	UpdateLoanStatusInTx(ctx context.Context, tx pgx.Tx, loanID int64, status LoanStatus) error

	CheckIfAllPaymentsMadeInTx(ctx context.Context, tx pgx.Tx, loanID int64) (bool, error)

	GetTotalOutstandingAmount(ctx context.Context, loanID int64) (float64, error)

	BeginTx(ctx context.Context) (pgx.Tx, error)

	CommitTx(ctx context.Context, tx pgx.Tx) error

	RollbackTx(ctx context.Context, tx pgx.Tx) error
}
