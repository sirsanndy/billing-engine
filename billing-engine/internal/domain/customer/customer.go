package customer

import "time"

type Customer struct {
	CustomerID   int64     `json:"customerId"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	IsDelinquent bool      `json:"isDelinquent"`
	Active       bool      `json:"active"`
	LoanID       *int64    `json:"loanId,omitempty"`
	CreateDate   time.Time `json:"createDate"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func NewCustomer(name, address string) *Customer {
	now := time.Now()
	return &Customer{

		Name:         name,
		Address:      address,
		IsDelinquent: false,
		Active:       true,
		LoanID:       nil,
		CreateDate:   now,
		UpdatedAt:    now,
	}
}

func (c *Customer) AssignLoan(loanID int64) {
	c.LoanID = &loanID
	c.UpdatedAt = time.Now()
}

func (c *Customer) SetDelinquencyStatus(isDelinquent bool) {
	if c.IsDelinquent != isDelinquent {
		c.IsDelinquent = isDelinquent
		c.UpdatedAt = time.Now()
	}
}

func (c *Customer) Deactivate() {
	if c.Active {
		c.Active = false
		c.UpdatedAt = time.Now()
	}
}
