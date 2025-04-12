package dto

import (
	"billing-engine/internal/domain/customer"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	validRequest = "Valid request"
)

func TestCreateCustomerRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request CreateCustomerRequest
		wantErr bool
	}{
		{validRequest, CreateCustomerRequest{Name: "John Doe", Address: "123 Street"}, false},
		{"Empty name", CreateCustomerRequest{Name: "", Address: "123 Street"}, true},
		{"Empty address", CreateCustomerRequest{Name: "John Doe", Address: ""}, true},
		{"Empty name and address", CreateCustomerRequest{Name: "", Address: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateCustomerAddressRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateCustomerAddressRequest
		wantErr bool
	}{
		{validRequest, UpdateCustomerAddressRequest{Address: "123 Street"}, false},
		{"Empty address", UpdateCustomerAddressRequest{Address: ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAssignLoanRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request AssignLoanRequest
		wantErr bool
	}{
		{validRequest, AssignLoanRequest{LoanID: 1}, false},
		{"Invalid loanId", AssignLoanRequest{LoanID: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateDelinquencyRequestValidate(t *testing.T) {
	request := UpdateDelinquencyRequest{IsDelinquent: true}
	err := request.Validate()
	assert.NoError(t, err)
}

func TestNewCustomerResponse(t *testing.T) {
	loanID := int64(123)
	cust := &customer.Customer{
		CustomerID:   1,
		Name:         "John Doe",
		Address:      "123 Street",
		IsDelinquent: false,
		Active:       true,
		LoanID:       &loanID,
		CreateDate:   time.Now(),
		UpdatedAt:    time.Now(),
	}

	resp := NewCustomerResponse(cust)
	assert.Equal(t, strconv.FormatInt(cust.CustomerID, 10), resp.CustomerID)
	assert.Equal(t, cust.Name, resp.Name)
	assert.Equal(t, cust.Address, resp.Address)
	assert.Equal(t, cust.IsDelinquent, resp.IsDelinquent)
	assert.Equal(t, cust.Active, resp.Active)
	assert.NotNil(t, resp.LoanID)
	assert.Equal(t, strconv.FormatInt(*cust.LoanID, 10), *resp.LoanID)
	assert.Equal(t, cust.CreateDate, resp.CreateDate)
	assert.Equal(t, cust.UpdatedAt, resp.UpdatedAt)

	resp = NewCustomerResponse(nil)
	assert.Equal(t, CustomerResponse{}, resp)
}
