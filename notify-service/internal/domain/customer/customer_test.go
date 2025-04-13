package customer

import (
	"testing"
	"time"
)

func TestCustomerStruct(t *testing.T) {
	now := time.Now()
	loanID := int64(12345)

	customer := Customer{
		CustomerID:   1,
		Name:         "John Doe",
		Address:      "123 Main St",
		IsDelinquent: false,
		Active:       true,
		LoanID:       &loanID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if customer.CustomerID != 1 {
		t.Errorf("expected ID to be 1, got %d", customer.CustomerID)
	}

	if customer.Name != "John Doe" {
		t.Errorf("expected Name to be 'John Doe', got %s", customer.Name)
	}

	if customer.Address != "123 Main St" {
		t.Errorf("expected Address to be '123 Main St', got %s", customer.Address)
	}

	if customer.IsDelinquent != false {
		t.Errorf("expected IsDelinquent to be false, got %v", customer.IsDelinquent)
	}

	if customer.Active != true {
		t.Errorf("expected Active to be true, got %v", customer.Active)
	}

	if customer.LoanID == nil || *customer.LoanID != loanID {
		t.Errorf("expected LoanID to be %d, got %v", loanID, customer.LoanID)
	}

	if !customer.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt to be %v, got %v", now, customer.CreatedAt)
	}

	if !customer.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt to be %v, got %v", now, customer.UpdatedAt)
	}
}
