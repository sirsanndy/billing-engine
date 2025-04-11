package dto

import (
	"billing-engine/internal/domain/customer"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CreateCustomerRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (r *CreateCustomerRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.TrimSpace(r.Address) == "" {
		return fmt.Errorf("address cannot be empty")
	}

	return nil
}

type UpdateCustomerAddressRequest struct {
	Address string `json:"address"`
}

func (r *UpdateCustomerAddressRequest) Validate() error {
	if strings.TrimSpace(r.Address) == "" {
		return fmt.Errorf("address cannot be empty")
	}
	return nil
}

type AssignLoanRequest struct {
	LoanID int64 `json:"loanId"`
}

func (r *AssignLoanRequest) Validate() error {
	if r.LoanID <= 0 {
		return fmt.Errorf("loanId must be a positive number")
	}
	return nil
}

type UpdateDelinquencyRequest struct {
	IsDelinquent bool `json:"isDelinquent"`
}

func (r *UpdateDelinquencyRequest) Validate() error {

	return nil
}

type CustomerResponse struct {
	CustomerID   string    `json:"customerId"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	IsDelinquent bool      `json:"isDelinquent"`
	Active       bool      `json:"active"`
	LoanID       *string   `json:"loanId,omitempty"`
	CreateDate   time.Time `json:"createDate"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewCustomerResponse(cust *customer.Customer) CustomerResponse {
	if cust == nil {

		return CustomerResponse{}
	}

	var loanIDStr *string

	if cust.LoanID != nil {
		s := strconv.FormatInt(*cust.LoanID, 10)
		loanIDStr = &s
	}

	return CustomerResponse{
		CustomerID:   strconv.FormatInt(cust.CustomerID, 10),
		Name:         cust.Name,
		Address:      cust.Address,
		IsDelinquent: cust.IsDelinquent,
		Active:       cust.Active,
		LoanID:       loanIDStr,
		CreateDate:   cust.CreateDate,
		UpdatedAt:    cust.UpdatedAt,
	}
}
