package customer_test

import (
	"billing-engine/internal/domain/customer"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCustomer(t *testing.T) {
	name := "Alice Wonderland"
	address := "123 Rabbit Hole Lane"
	timeBefore := time.Now()

	cust := customer.NewCustomer(name, address)
	timeAfter := time.Now()

	assert.NotNil(t, cust, "NewCustomer should return a non-nil customer")

	assert.Equal(t, name, cust.Name, "Customer name should match input")
	assert.Equal(t, address, cust.Address, "Customer address should match input")
	assert.False(t, cust.IsDelinquent, "New customer should not be delinquent")
	assert.True(t, cust.Active, "New customer should be active")
	assert.Nil(t, cust.LoanID, "New customer should have nil LoanID")

	assert.False(t, cust.CreateDate.IsZero(), "CreateDate should be set")
	assert.False(t, cust.UpdatedAt.IsZero(), "UpdatedAt should be set")
	assert.Equal(t, cust.CreateDate, cust.UpdatedAt, "CreateDate and UpdatedAt should initially be the same")

	assert.True(t, !cust.CreateDate.Before(timeBefore) && !cust.CreateDate.After(timeAfter), "CreateDate should be around the time of creation")
	assert.True(t, !cust.UpdatedAt.Before(timeBefore) && !cust.UpdatedAt.After(timeAfter), "UpdatedAt should be around the time of creation")

	assert.Equal(t, int64(0), cust.CustomerID, "CustomerID should be initialized to 0")
}

func TestCustomer_AssignLoan(t *testing.T) {
	cust := customer.NewCustomer("Bob The Builder", "Fixit Town")
	initialUpdateTime := cust.UpdatedAt
	loanID := int64(101)

	time.Sleep(1 * time.Millisecond)

	cust.AssignLoan(loanID)

	assert.NotNil(t, cust.LoanID, "LoanID should not be nil after assignment")
	if cust.LoanID != nil {
		assert.Equal(t, loanID, *cust.LoanID, "Assigned LoanID should match the provided ID")
	}
	assert.True(t, cust.UpdatedAt.After(initialUpdateTime), "UpdatedAt should be updated after assigning loan")
}

func TestCustomer_SetDelinquencyStatus(t *testing.T) {
	t.Run("Set delinquent from false to true", func(t *testing.T) {
		cust := customer.NewCustomer("Charlie Chaplin", "Hollywood")
		initialUpdateTime := cust.UpdatedAt
		assert.False(t, cust.IsDelinquent, "Customer should initially not be delinquent")

		time.Sleep(1 * time.Millisecond)
		cust.SetDelinquencyStatus(true)

		assert.True(t, cust.IsDelinquent, "Customer should now be delinquent")
		assert.True(t, cust.UpdatedAt.After(initialUpdateTime), "UpdatedAt should be updated")
	})

	t.Run("Set non-delinquent from true to false", func(t *testing.T) {
		cust := customer.NewCustomer("Diana Prince", "Themyscira")
		cust.IsDelinquent = true
		initialUpdateTime := time.Now()
		cust.UpdatedAt = initialUpdateTime

		time.Sleep(1 * time.Millisecond)
		cust.SetDelinquencyStatus(false)

		assert.False(t, cust.IsDelinquent, "Customer should now not be delinquent")
		assert.True(t, cust.UpdatedAt.After(initialUpdateTime), "UpdatedAt should be updated")
	})

	t.Run("Set delinquent to true when already true", func(t *testing.T) {
		cust := customer.NewCustomer("Eric Cartman", "South Park")
		cust.IsDelinquent = true
		initialUpdateTime := time.Now()
		cust.UpdatedAt = initialUpdateTime

		time.Sleep(1 * time.Millisecond)
		cust.SetDelinquencyStatus(true)

		assert.True(t, cust.IsDelinquent, "Customer should remain delinquent")

		assert.Equal(t, initialUpdateTime, cust.UpdatedAt, "UpdatedAt should NOT be updated")
	})

	t.Run("Set delinquent to false when already false", func(t *testing.T) {
		cust := customer.NewCustomer("Fiona Gallagher", "Chicago")
		initialUpdateTime := cust.UpdatedAt
		assert.False(t, cust.IsDelinquent, "Customer should initially not be delinquent")

		time.Sleep(1 * time.Millisecond)
		cust.SetDelinquencyStatus(false)

		assert.False(t, cust.IsDelinquent, "Customer should remain non-delinquent")

		assert.Equal(t, initialUpdateTime, cust.UpdatedAt, "UpdatedAt should NOT be updated")
	})
}

func TestCustomer_Deactivate(t *testing.T) {
	t.Run("Deactivate an active customer", func(t *testing.T) {
		cust := customer.NewCustomer("Gandalf Grey", "Middle Earth")
		initialUpdateTime := cust.UpdatedAt
		assert.True(t, cust.Active, "Customer should initially be active")

		time.Sleep(1 * time.Millisecond)
		cust.Deactivate()

		assert.False(t, cust.Active, "Customer should now be inactive")
		assert.True(t, cust.UpdatedAt.After(initialUpdateTime), "UpdatedAt should be updated")
	})

	t.Run("Deactivate an already inactive customer", func(t *testing.T) {
		cust := customer.NewCustomer("Harry Potter", "Privet Drive")
		cust.Active = false
		initialUpdateTime := time.Now()
		cust.UpdatedAt = initialUpdateTime

		time.Sleep(1 * time.Millisecond)
		cust.Deactivate()

		assert.False(t, cust.Active, "Customer should remain inactive")

		assert.Equal(t, initialUpdateTime, cust.UpdatedAt, "UpdatedAt should NOT be updated")
	})
}
