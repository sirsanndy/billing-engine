package event

import (
	"context"
	"time"
)

type CustomerEventPayload struct {
	CustomerID   int64     `json:"customerId"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	IsDelinquent bool      `json:"isDelinquent"`
	Active       bool      `json:"active"`
	LoanID       *int64    `json:"loanId,omitempty"`
	CreateDate   time.Time `json:"createDate"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CustomerCreatedEvent struct {
	Timestamp time.Time            `json:"timestamp"`
	Payload   CustomerEventPayload `json:"payload"`
}

type CustomerUpdatedEvent struct {
	Timestamp time.Time            `json:"timestamp"`
	Payload   CustomerEventPayload `json:"payload"`
}

func (p *RabbitMQEventPublisher) PublishCustomerCreated(ctx context.Context, event CustomerCreatedEvent) error {
	return p.publish(ctx, routingKeyCustomerCreated, event)
}

func (p *RabbitMQEventPublisher) PublishCustomerUpdated(ctx context.Context, event CustomerUpdatedEvent) error {
	return p.publish(ctx, routingKeyCustomerUpdated, event)
}

var _ EventPublisher = (*RabbitMQEventPublisher)(nil)
