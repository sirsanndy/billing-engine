package loan

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLoan(t *testing.T) {
	t.Run("should error when inputs are invalid", func(t *testing.T) {
		loan, err := NewLoan(-1, -1, -1, time.Time{})
		assert.Error(t, err)
		assert.Nil(t, loan)
	})

	t.Run("should create a loan with provided values", func(t *testing.T) {
		startDate := time.Now()
		loan, err := NewLoan(1_000_000, 52, 0.05, startDate)
		assert.NoError(t, err)
		assert.NotNil(t, loan)
		assert.Equal(t, 1_000_000.0, loan.PrincipalAmount)
		assert.Equal(t, 52, loan.TermWeeks)
		assert.Equal(t, 0.05, loan.InterestRate)
		assert.Equal(t, StatusActive, loan.Status)
		assert.Equal(t, startDate, loan.StartDate)
		assert.Equal(t, roundTo(1_000_000*1.05, 2), loan.TotalLoanAmount)
		assert.Equal(t, roundTo((1_000_000*1.05)/52, 2), loan.WeeklyPaymentAmount)
	})

	t.Run("should return error for invalid term weeks", func(t *testing.T) {
		_, err := NewLoan(1_000_000, 0, 0.05, time.Now())
		assert.Error(t, err)
	})
}

func TestGenerateSchedule(t *testing.T) {
	t.Run("should generate a valid payment schedule", func(t *testing.T) {
		startDate := time.Now()
		loan, err := NewLoan(1_000_000, 10, 0.1, startDate)
		assert.NoError(t, err)

		schedule, err := loan.GenerateSchedule()
		assert.NoError(t, err)
		assert.Len(t, schedule, 10)

		accumulatedPayment := 0.0
		for i, entry := range schedule {
			assert.Equal(t, i+1, entry.WeekNumber)
			assert.Equal(t, startDate.AddDate(0, 0, (i+1)*7), entry.DueDate)
			assert.Equal(t, PaymentStatusPending, entry.Status)
			accumulatedPayment += entry.DueAmount
		}

		assert.InDelta(t, loan.TotalLoanAmount, accumulatedPayment, 0.01)
	})

	t.Run("should return error for invalid loan terms", func(t *testing.T) {
		loan := &Loan{
			TermWeeks:           0,
			WeeklyPaymentAmount: -1,
		}
		_, err := loan.GenerateSchedule()
		assert.Error(t, err)
	})

	t.Run("should handle rounding issues in the last payment", func(t *testing.T) {
		startDate := time.Now()
		loan, err := NewLoan(1_000_003, 3, 0.0, startDate)
		assert.NoError(t, err)

		schedule, err := loan.GenerateSchedule()
		assert.NoError(t, err)
		assert.Len(t, schedule, 3)

		accumulatedPayment := 0.0
		for _, entry := range schedule {
			accumulatedPayment += entry.DueAmount
		}

		assert.InDelta(t, loan.TotalLoanAmount, accumulatedPayment, 0.01)
	})
}
