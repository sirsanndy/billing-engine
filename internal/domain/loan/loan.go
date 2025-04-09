package loan

import (
	"billing-engine/internal/pkg/apperrors"
	"fmt"
	"math"
	"time"
)

const (
	DefaultPrincipal    = 5_000_000.0
	DefaultTermWeeks    = 50
	DefaultInterestRate = 0.10
)

type LoanStatus string

const (
	StatusActive     LoanStatus = "ACTIVE"
	StatusPaidOff    LoanStatus = "PAID_OFF"
	StatusDelinquent LoanStatus = "DELINQUENT"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "PENDING"
	PaymentStatusPaid    PaymentStatus = "PAID"
	PaymentStatusMissed  PaymentStatus = "MISSED"
)

type Loan struct {
	ID                  int64
	PrincipalAmount     float64
	InterestRate        float64
	TermWeeks           int
	WeeklyPaymentAmount float64
	TotalLoanAmount     float64
	StartDate           time.Time
	Status              LoanStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Schedule            []ScheduleEntry
}

type ScheduleEntry struct {
	ID          int64
	LoanID      int64
	WeekNumber  int
	DueDate     time.Time
	DueAmount   float64
	PaidAmount  float64
	PaymentDate *time.Time
	Status      PaymentStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewLoan(principal float64, termWeeks int, annualInterestRate float64, startDate time.Time) (*Loan, error) {
	if principal <= 0 {
		principal = DefaultPrincipal
	}
	if termWeeks <= 0 {
		termWeeks = DefaultTermWeeks
	}

	if annualInterestRate < 0 {
		annualInterestRate = DefaultInterestRate
	}
	if termWeeks <= 0 {
		return nil, fmt.Errorf("%w: term weeks must be positive", apperrors.ErrInvalidArgument)
	}
	if startDate.IsZero() {

		startDate = time.Now().Truncate(24 * time.Hour)
	}

	loan := &Loan{
		PrincipalAmount: principal,
		TermWeeks:       termWeeks,
		InterestRate:    annualInterestRate,
		StartDate:       startDate,
		Status:          StatusActive,
	}

	totalInterest := loan.PrincipalAmount * loan.InterestRate
	loan.TotalLoanAmount = loan.PrincipalAmount + totalInterest

	loan.WeeklyPaymentAmount = roundTo(loan.TotalLoanAmount/float64(loan.TermWeeks), 2)

	return loan, nil
}

func (l *Loan) GenerateSchedule() ([]ScheduleEntry, error) {
	if l.TermWeeks <= 0 || l.WeeklyPaymentAmount < 0 {
		return nil, fmt.Errorf("%w: invalid loan terms for schedule generation", apperrors.ErrInvalidArgument)
	}

	schedule := make([]ScheduleEntry, 0, l.TermWeeks)
	currentDueDate := l.StartDate
	accumulatedPayment := 0.0

	for week := 1; week <= l.TermWeeks; week++ {

		currentDueDate = l.StartDate.AddDate(0, 0, week*7)

		paymentAmount := l.WeeklyPaymentAmount
		if week == l.TermWeeks {

			paymentAmount = roundTo(l.TotalLoanAmount-accumulatedPayment, 2)
			if paymentAmount < 0 {
				paymentAmount = 0
			}
		}

		entry := ScheduleEntry{

			WeekNumber: week,
			DueDate:    currentDueDate,
			DueAmount:  paymentAmount,
			Status:     PaymentStatusPending,
		}
		schedule = append(schedule, entry)
		accumulatedPayment += paymentAmount
	}

	finalAccumulated := roundTo(accumulatedPayment, 2)
	expectedTotal := roundTo(l.TotalLoanAmount, 2)
	if math.Abs(finalAccumulated-expectedTotal) > 0.01 {
		return nil, fmt.Errorf("%w: schedule generation failed sanity check - total payment %.2f != expected total %.2f",
			apperrors.ErrInternalServer, finalAccumulated, expectedTotal)
	}

	return schedule, nil
}

func roundTo(n float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(n*pow) / pow
}
