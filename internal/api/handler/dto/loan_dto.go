package dto

import (
	"billing-engine/internal/domain/loan"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

type CreateLoanRequest struct {
	CustomerID         int64   `json:"customerId"`
	Principal          float64 `json:"principal"`
	TermWeeks          int     `json:"termWeeks"`
	AnnualInterestRate float64 `json:"annualInterestRate"`
	StartDate          string  `json:"startDate"`
}

func (r *CreateLoanRequest) Validate() error {
	if r.Principal <= 0 {
		return fmt.Errorf("principal must be grater than zero")
	}
	if r.AnnualInterestRate <= 0 {
		return fmt.Errorf("annualInterestRate must be greater than zero")
	}
	if r.TermWeeks <= 0 {
		return fmt.Errorf("termWeeks must be positive")
	}
	if _, err := time.Parse(time.RFC3339[:10], r.StartDate); err != nil || r.StartDate == "" {
		return fmt.Errorf("invalid startDate format (use YYYY-MM-DD): %w", err)
	}
	return nil
}

type MakePaymentRequest struct {
	Amount string `json:"amount"`
}

func (r *MakePaymentRequest) Validate() error {
	if _, err := decimal.NewFromString(r.Amount); err != nil || r.Amount == "" {
		return fmt.Errorf("invalid payment amount: %w", err)
	}
	return nil
}

type LoanResponse struct {
	ID                  string                  `json:"id"`
	PrincipalAmount     string                  `json:"principalAmount"`
	InterestRate        string                  `json:"interestRate"`
	TermWeeks           int                     `json:"termWeeks"`
	WeeklyPaymentAmount string                  `json:"weeklyPaymentAmount"`
	TotalLoanAmount     string                  `json:"totalLoanAmount"`
	StartDate           string                  `json:"startDate"`
	Status              string                  `json:"status"`
	CreatedAt           time.Time               `json:"createdAt"`
	UpdatedAt           time.Time               `json:"updatedAt"`
	Schedule            []ScheduleEntryResponse `json:"schedule,omitempty"`
}

type ScheduleEntryResponse struct {
	ID          string     `json:"id"`
	WeekNumber  int        `json:"weekNumber"`
	DueDate     string     `json:"dueDate"`
	DueAmount   string     `json:"dueAmount"`
	PaidAmount  *string    `json:"paidAmount,omitempty"`
	PaymentDate *time.Time `json:"paymentDate,omitempty"`
	Status      string     `json:"status"`
}

type OutstandingResponse struct {
	LoanID            string `json:"loanId"`
	OutstandingAmount string `json:"outstandingAmount"`
}

type DelinquentResponse struct {
	LoanID       string `json:"loanId"`
	IsDelinquent bool   `json:"isDelinquent"`
}

type ErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type TokenRequest struct {
	Username string `json:"username"`
}

func NewLoanResponse(domainLoan *loan.Loan, includeSchedule bool) LoanResponse {
	formatDecimalMoney := func(d decimal.Decimal) string {
		return d.StringFixed(2)
	}

	principalStr := formatDecimalMoney(decimal.NewFromFloat(domainLoan.PrincipalAmount))
	weeklyPaymentStr := formatDecimalMoney(decimal.NewFromFloat(domainLoan.WeeklyPaymentAmount))
	totalLoanStr := formatDecimalMoney(decimal.NewFromFloat(domainLoan.TotalLoanAmount))
	interestRateStr := decimal.NewFromFloat(domainLoan.InterestRate).String()

	resp := LoanResponse{
		ID:                  strconv.FormatInt(domainLoan.ID, 10),
		PrincipalAmount:     principalStr,
		InterestRate:        interestRateStr,
		TermWeeks:           domainLoan.TermWeeks,
		WeeklyPaymentAmount: weeklyPaymentStr,
		TotalLoanAmount:     totalLoanStr,
		StartDate:           domainLoan.StartDate.Format(time.RFC3339[:10]),
		Status:              string(domainLoan.Status),
		CreatedAt:           domainLoan.CreatedAt,
		UpdatedAt:           domainLoan.UpdatedAt,
	}

	if includeSchedule && domainLoan.Schedule != nil {
		resp.Schedule = make([]ScheduleEntryResponse, len(domainLoan.Schedule))
		for i, entry := range domainLoan.Schedule {
			resp.Schedule[i] = NewScheduleEntryResponse(&entry)
		}
	}

	return resp
}

func NewScheduleEntryResponse(entry *loan.ScheduleEntry) ScheduleEntryResponse {
	formatDecimalMoney := func(d decimal.Decimal) string {
		return d.StringFixed(2)
	}

	var paidAmountStr *string
	if entry.PaidAmount != 0 {
		pd := decimal.NewFromFloat(entry.PaidAmount)
		s := formatDecimalMoney(pd)
		paidAmountStr = &s
	}

	return ScheduleEntryResponse{
		ID:          strconv.FormatInt(entry.ID, 10),
		WeekNumber:  entry.WeekNumber,
		DueDate:     entry.DueDate.Format(time.RFC3339[:10]),
		DueAmount:   formatDecimalMoney(decimal.NewFromFloat(entry.DueAmount)),
		PaidAmount:  paidAmountStr,
		PaymentDate: entry.PaymentDate,
		Status:      string(entry.Status),
	}
}
