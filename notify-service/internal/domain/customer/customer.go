package customer

import "time"

type Customer struct {
	CustomerID   int64
	Name         string
	Address      string
	IsDelinquent bool
	Active       bool
	LoanID       *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
