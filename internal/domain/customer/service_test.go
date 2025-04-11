package customer_test

import (
	"billing-engine/internal/domain/customer"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTest() (*customer.MockCustomerRepository, customer.CustomerService) {
	mockRepo := new(customer.MockCustomerRepository)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := customer.NewCustomerService(mockRepo, logger)
	return mockRepo, service
}

func TestCustomerService_CreateNewCustomer(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		name := "   Test User  "
		address := " 123 Test St "
		expectedName := "Test User"
		expectedAddress := "123 Test St"
		expectedCustomerID := int64(1)

		mockRepo.On("Save", ctx, mock.MatchedBy(func(c *customer.Customer) bool {

			match := assert.ObjectsAreEqual(&customer.Customer{
				Name:         expectedName,
				Address:      expectedAddress,
				IsDelinquent: false,
				Active:       true,
				LoanID:       nil,
			}, &customer.Customer{
				Name:         c.Name,
				Address:      c.Address,
				IsDelinquent: c.IsDelinquent,
				Active:       c.Active,
				LoanID:       c.LoanID,
			})
			if match {

				c.CustomerID = expectedCustomerID
				c.CreateDate = time.Now()
				c.UpdatedAt = c.CreateDate
			}
			return match
		})).Return(nil).Once()

		createdCustomer, err := service.CreateNewCustomer(ctx, name, address)

		assert.NoError(t, err)
		assert.NotNil(t, createdCustomer)
		if createdCustomer != nil {
			assert.Equal(t, expectedCustomerID, createdCustomer.CustomerID)
			assert.Equal(t, expectedName, createdCustomer.Name)
			assert.Equal(t, expectedAddress, createdCustomer.Address)
			assert.True(t, createdCustomer.Active)
			assert.False(t, createdCustomer.IsDelinquent)
			assert.Nil(t, createdCustomer.LoanID)
			assert.False(t, createdCustomer.CreateDate.IsZero())
			assert.Equal(t, createdCustomer.CreateDate, createdCustomer.UpdatedAt)
		}
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Empty Name", func(t *testing.T) {
		mockRepo, service := setupTest()
		_, err := service.CreateNewCustomer(ctx, "", "Some Address")
		assert.Error(t, err)
		assert.EqualError(t, err, "customer name cannot be empty")
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Empty Address", func(t *testing.T) {
		mockRepo, service := setupTest()
		_, err := service.CreateNewCustomer(ctx, "Some Name", "  ")
		assert.Error(t, err)
		assert.EqualError(t, err, "customer address cannot be empty")
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Repository Save Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("database connection failed")

		mockRepo.On("Save", ctx, mock.AnythingOfType("*customer.Customer")).Return(dbError).Once()

		createdCustomer, err := service.CreateNewCustomer(ctx, "Valid Name", "Valid Address")

		assert.Error(t, err)
		assert.Nil(t, createdCustomer)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), "failed to save new customer")
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_GetCustomer(t *testing.T) {
	ctx := context.Background()
	customerID := int64(42)

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		expectedCustomer := &customer.Customer{CustomerID: customerID, Name: "Test", Active: true}

		mockRepo.On("FindByID", ctx, customerID).Return(expectedCustomer, nil).Once()

		cust, err := service.GetCustomer(ctx, customerID)

		assert.NoError(t, err)
		assert.Equal(t, expectedCustomer, cust)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, service := setupTest()

		mockRepo.On("FindByID", ctx, customerID).Return(nil, customer.ErrNotFound).Once()

		cust, err := service.GetCustomer(ctx, customerID)

		assert.Error(t, err)
		assert.Nil(t, cust)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("internal server error")

		mockRepo.On("FindByID", ctx, customerID).Return(nil, dbError).Once()

		cust, err := service.GetCustomer(ctx, customerID)

		assert.Error(t, err)
		assert.Nil(t, cust)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to get customer %d", customerID))
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_ListActiveCustomers(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		expectedCustomers := []*customer.Customer{
			{CustomerID: 1, Name: "Alice", Active: true},
			{CustomerID: 2, Name: "Bob", Active: true},
		}

		mockRepo.On("FindAll", ctx, true).Return(expectedCustomers, nil).Once()

		customers, err := service.ListActiveCustomers(ctx)

		assert.NoError(t, err)
		assert.Equal(t, expectedCustomers, customers)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Empty List", func(t *testing.T) {
		mockRepo, service := setupTest()
		expectedCustomers := []*customer.Customer{}

		mockRepo.On("FindAll", ctx, true).Return(expectedCustomers, nil).Once()

		customers, err := service.ListActiveCustomers(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, customers)
		assert.Empty(t, customers)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("query failed")

		mockRepo.On("FindAll", ctx, true).Return(nil, dbError).Once()

		customers, err := service.ListActiveCustomers(ctx)

		assert.Error(t, err)
		assert.Nil(t, customers)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), "failed to list active customers")
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_UpdateCustomerAddress(t *testing.T) {
	ctx := context.Background()
	customerID := int64(55)
	oldAddress := "Old Address Lane"
	newAddress := "  New Address Ave  "
	trimmedNewAddress := "New Address Ave"

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Update Me", Address: oldAddress, Active: true}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		mockRepo.On("Save", ctx, mock.MatchedBy(func(c *customer.Customer) bool {
			return c.CustomerID == customerID && c.Address == trimmedNewAddress
		})).Return(nil).Once()

		err := service.UpdateCustomerAddress(ctx, customerID, newAddress)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - No Change Needed", func(t *testing.T) {
		mockRepo, service := setupTest()

		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Update Me", Address: trimmedNewAddress, Active: true}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		err := service.UpdateCustomerAddress(ctx, customerID, " "+trimmedNewAddress+" ")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Empty New Address", func(t *testing.T) {
		mockRepo, service := setupTest()
		err := service.UpdateCustomerAddress(ctx, customerID, "   ")
		assert.Error(t, err)
		assert.EqualError(t, err, "new address cannot be empty")
		mockRepo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - FindByID Not Found", func(t *testing.T) {
		mockRepo, service := setupTest()
		mockRepo.On("FindByID", ctx, customerID).Return(nil, customer.ErrNotFound).Once()

		err := service.UpdateCustomerAddress(ctx, customerID, newAddress)

		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - FindByID Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("find failed")
		mockRepo.On("FindByID", ctx, customerID).Return(nil, dbError).Once()

		err := service.UpdateCustomerAddress(ctx, customerID, newAddress)

		assert.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("cannot find customer %d to update address", customerID))
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Save Not Found (Race Condition)", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Update Me", Address: oldAddress, Active: true}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		mockRepo.On("Save", ctx, mock.AnythingOfType("*customer.Customer")).Return(customer.ErrNotFound).Once()

		err := service.UpdateCustomerAddress(ctx, customerID, newAddress)

		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Save Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Update Me", Address: oldAddress, Active: true}
		dbError := errors.New("save conflict")

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		mockRepo.On("Save", ctx, mock.AnythingOfType("*customer.Customer")).Return(dbError).Once()

		err := service.UpdateCustomerAddress(ctx, customerID, newAddress)

		assert.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to save updated address for customer %d", customerID))
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_AssignLoanToCustomer(t *testing.T) {
	ctx := context.Background()
	customerID := int64(77)
	loanID := int64(1001)
	differentLoanID := int64(999)

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: true, LoanID: nil}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		mockRepo.On("Save", ctx, mock.MatchedBy(func(c *customer.Customer) bool {
			return c.CustomerID == customerID && c.LoanID != nil && *c.LoanID == loanID
		})).Return(nil).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Loan Already Assigned (Same ID)", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: true, LoanID: &loanID}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Invalid Loan ID", func(t *testing.T) {
		mockRepo, service := setupTest()
		err := service.AssignLoanToCustomer(ctx, customerID, 0)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid loan ID provided")
		mockRepo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)

		err = service.AssignLoanToCustomer(ctx, customerID, -10)
		assert.Error(t, err)
		assert.EqualError(t, err, "invalid loan ID provided")
	})

	t.Run("Error - FindByID Not Found", func(t *testing.T) {
		mockRepo, service := setupTest()
		mockRepo.On("FindByID", ctx, customerID).Return(nil, customer.ErrNotFound).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - FindByID Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("find failed")
		mockRepo.On("FindByID", ctx, customerID).Return(nil, dbError).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("cannot find customer %d to assign loan", customerID))
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Customer Inactive", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: false, LoanID: nil}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("cannot assign loan to inactive customer %d", customerID))
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Customer Already Has Different Loan", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: true, LoanID: &differentLoanID}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("customer %d already assigned loan %d", customerID, differentLoanID))
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	})

	t.Run("Error - Save Duplicate Loan ID", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: true, LoanID: nil}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("*customer.Customer")).Return(customer.ErrDuplicateLoanID).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrDuplicateLoanID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Save Not Found (Race Condition)", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: true, LoanID: nil}

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("*customer.Customer")).Return(customer.ErrNotFound).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Save Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		existingCustomer := &customer.Customer{CustomerID: customerID, Name: "Assign Loan", Active: true, LoanID: nil}
		dbError := errors.New("save failed")

		mockRepo.On("FindByID", ctx, customerID).Return(existingCustomer, nil).Once()
		mockRepo.On("Save", ctx, mock.AnythingOfType("*customer.Customer")).Return(dbError).Once()

		err := service.AssignLoanToCustomer(ctx, customerID, loanID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to save loan assignment for customer %d", customerID))
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_UpdateDelinquency(t *testing.T) {
	ctx := context.Background()
	customerID := int64(88)

	testCases := []struct {
		name         string
		isDelinquent bool
		repoError    error
		expectError  bool
		expectedErr  error
	}{
		{"Success - Set Delinquent", true, nil, false, nil},
		{"Success - Set Not Delinquent", false, nil, false, nil},
		{"Error - Not Found", true, customer.ErrNotFound, true, customer.ErrNotFound},
		{"Error - Repository Failure", false, errors.New("db update failed"), true, errors.New("db update failed")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo, service := setupTest()

			mockRepo.On("SetDelinquencyStatus", ctx, customerID, tc.isDelinquent).Return(tc.repoError).Once()

			err := service.UpdateDelinquency(ctx, customerID, tc.isDelinquent)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErr != nil {

					if errors.Is(tc.repoError, customer.ErrNotFound) {
						assert.ErrorIs(t, err, tc.expectedErr)
					} else {
						assert.ErrorContains(t, err, tc.expectedErr.Error())
						assert.ErrorContains(t, err, fmt.Sprintf("failed to update delinquency for customer %d", customerID))
					}
				}
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCustomerService_DeactivateCustomer(t *testing.T) {
	ctx := context.Background()
	customerID := int64(99)

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		mockRepo.On("SetActiveStatus", ctx, customerID, false).Return(nil).Once()
		err := service.DeactivateCustomer(ctx, customerID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, service := setupTest()
		mockRepo.On("SetActiveStatus", ctx, customerID, false).Return(customer.ErrNotFound).Once()
		err := service.DeactivateCustomer(ctx, customerID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("update failed")
		mockRepo.On("SetActiveStatus", ctx, customerID, false).Return(dbError).Once()
		err := service.DeactivateCustomer(ctx, customerID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to deactivate customer %d", customerID))
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_ReactivateCustomer(t *testing.T) {
	ctx := context.Background()
	customerID := int64(111)

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		mockRepo.On("SetActiveStatus", ctx, customerID, true).Return(nil).Once()
		err := service.ReactivateCustomer(ctx, customerID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, service := setupTest()
		mockRepo.On("SetActiveStatus", ctx, customerID, true).Return(customer.ErrNotFound).Once()
		err := service.ReactivateCustomer(ctx, customerID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("update failed")
		mockRepo.On("SetActiveStatus", ctx, customerID, true).Return(dbError).Once()
		err := service.ReactivateCustomer(ctx, customerID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to reactivate customer %d", customerID))
		mockRepo.AssertExpectations(t)
	})
}

func TestCustomerService_FindCustomerByLoan(t *testing.T) {
	ctx := context.Background()
	loanID := int64(2002)
	customerID := int64(121)

	t.Run("Success", func(t *testing.T) {
		mockRepo, service := setupTest()
		expectedCustomer := &customer.Customer{CustomerID: customerID, Name: "Found By Loan", Active: true, LoanID: &loanID}

		mockRepo.On("FindByLoanID", ctx, loanID).Return(expectedCustomer, nil).Once()

		cust, err := service.FindCustomerByLoan(ctx, loanID)

		assert.NoError(t, err)
		assert.Equal(t, expectedCustomer, cust)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockRepo, service := setupTest()

		mockRepo.On("FindByLoanID", ctx, loanID).Return(nil, customer.ErrNotFound).Once()

		cust, err := service.FindCustomerByLoan(ctx, loanID)

		assert.Error(t, err)
		assert.Nil(t, cust)
		assert.ErrorIs(t, err, customer.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error - Repository Failure", func(t *testing.T) {
		mockRepo, service := setupTest()
		dbError := errors.New("internal server error")

		mockRepo.On("FindByLoanID", ctx, loanID).Return(nil, dbError).Once()

		cust, err := service.FindCustomerByLoan(ctx, loanID)

		assert.Error(t, err)
		assert.Nil(t, cust)
		assert.ErrorIs(t, err, dbError)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to find customer by loan ID %d", loanID))
		mockRepo.AssertExpectations(t)
	})
}

func TestNewCustomerService(t *testing.T) {
	t.Run("Panic on nil repository", func(t *testing.T) {
		assert.PanicsWithValue(t, "customer repository cannot be nil", func() {
			customer.NewCustomerService(nil, slog.Default())
		})
	})

	t.Run("Default logger if none provided", func(t *testing.T) {

		assert.NotPanics(t, func() {
			_ = customer.NewCustomerService(new(customer.MockCustomerRepository), nil)
		})

	})
}
