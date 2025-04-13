package dto

import (
	"billing-engine/internal/domain/loan"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLoanResponse(t *testing.T) {
	mockLoan := &loan.Loan{
		ID:                  1,
		PrincipalAmount:     1000.0,
		InterestRate:        5.0,
		TermWeeks:           10,
		WeeklyPaymentAmount: 105.0,
		TotalLoanAmount:     1050.0,
		StartDate:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:              loan.StatusActive,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		Schedule: []loan.ScheduleEntry{
			{
				ID:          1,
				WeekNumber:  1,
				DueDate:     time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC),
				DueAmount:   105.0,
				PaidAmount:  50.0,
				PaymentDate: nil,
				Status:      loan.PaymentStatusPaid,
			},
		},
	}

	t.Run("Test without schedule", func(t *testing.T) {
		response := NewLoanResponse(mockLoan, false)

		assert.Equal(t, "1", response.ID)
		assert.Equal(t, "1000.00", response.PrincipalAmount)
		assert.Equal(t, "5", response.InterestRate)
		assert.Equal(t, 10, response.TermWeeks)
		assert.Equal(t, "105.00", response.WeeklyPaymentAmount)
		assert.Equal(t, "1050.00", response.TotalLoanAmount)
		assert.Equal(t, "2023-01-01", response.StartDate)
		assert.Equal(t, string(loan.StatusActive), response.Status)
		assert.Equal(t, mockLoan.CreatedAt, response.CreatedAt)
		assert.Equal(t, mockLoan.UpdatedAt, response.UpdatedAt)
		assert.Nil(t, response.Schedule)
	})

	t.Run("Test with schedule", func(t *testing.T) {
		response := NewLoanResponse(mockLoan, true)

		assert.Equal(t, "1", response.ID)
		assert.Equal(t, "1000.00", response.PrincipalAmount)
		assert.Equal(t, "5", response.InterestRate)
		assert.Equal(t, 10, response.TermWeeks)
		assert.Equal(t, "105.00", response.WeeklyPaymentAmount)
		assert.Equal(t, "1050.00", response.TotalLoanAmount)
		assert.Equal(t, "2023-01-01", response.StartDate)
		assert.Equal(t, string(loan.StatusActive), response.Status)
		assert.Equal(t, mockLoan.CreatedAt, response.CreatedAt)
		assert.Equal(t, mockLoan.UpdatedAt, response.UpdatedAt)

		assert.NotNil(t, response.Schedule)
		assert.Len(t, response.Schedule, 1)

		scheduleEntry := response.Schedule[0]
		assert.Equal(t, "1", scheduleEntry.ID)
		assert.Equal(t, 1, scheduleEntry.WeekNumber)
		assert.Equal(t, "2023-01-08", scheduleEntry.DueDate)
		assert.Equal(t, "105.00", scheduleEntry.DueAmount)
		assert.NotNil(t, scheduleEntry.PaidAmount)
		assert.Equal(t, "50.00", *scheduleEntry.PaidAmount)
		assert.Nil(t, scheduleEntry.PaymentDate)
		assert.Equal(t, string(loan.PaymentStatusPaid), scheduleEntry.Status)
	})
}
