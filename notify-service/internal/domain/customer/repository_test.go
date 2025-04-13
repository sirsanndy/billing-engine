package customer

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

type mockCustomerRepository struct {
	data map[string]*Customer
}

func (m *mockCustomerRepository) Upsert(ctx context.Context, cust *Customer) error {
	if cust == nil {
		return errors.New("customer is nil")
	}
	if _, exists := m.data[strconv.FormatInt(cust.CustomerID, 10)]; exists {
		m.data[strconv.FormatInt(cust.CustomerID, 10)] = cust
		return nil
	}
	m.data[strconv.FormatInt(cust.CustomerID, 10)] = cust
	return nil
}

func TestCustomerRepositoryUpsert(t *testing.T) {
	mockRepo := &mockCustomerRepository{
		data: make(map[string]*Customer),
	}

	ctx := context.Background()

	t.Run("successfully upsert a new customer", func(t *testing.T) {
		customer := &Customer{CustomerID: 1, Name: "John Doe"}
		err := mockRepo.Upsert(ctx, customer)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if mockRepo.data["1"] != customer {
			t.Fatalf("expected customer to be added to repository")
		}
	})

	t.Run("successfully update an existing customer", func(t *testing.T) {
		customer := &Customer{CustomerID: 1, Name: "Jane Doe"}
		err := mockRepo.Upsert(ctx, customer)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if mockRepo.data["1"].Name != "Jane Doe" {
			t.Fatalf("expected customer name to be updated")
		}
	})

	t.Run("fail to upsert nil customer", func(t *testing.T) {
		err := mockRepo.Upsert(ctx, nil)
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}
	})
}
