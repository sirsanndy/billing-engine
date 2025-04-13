package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	routingKeyCustomerCreated = "customer.created"
	routingKeyCustomerUpdated = "customer.updated"
	publisherAppID            = "billing-engine"
)

type RabbitMQEventPublisher struct {
	conn         *amqp.Connection
	exchangeName string
	logger       *slog.Logger
}

type EventPublisher interface {
	PublishCustomerDelinquencyChanged(ctx context.Context, event CustomerDelinquencyChangedEvent) error
	PublishCustomerCreated(ctx context.Context, event CustomerCreatedEvent) error
	PublishCustomerUpdated(ctx context.Context, event CustomerUpdatedEvent) error
}

type CustomerDelinquencyChangedEvent struct {
	CustomerID int64     `json:"customerId"`
	LoanID     *int64    `json:"loanId,omitempty"`
	NewStatus  bool      `json:"newStatus"`
	OldStatus  bool      `json:"oldStatus"`
	Timestamp  time.Time `json:"timestamp"`
}

func NewRabbitMQEventPublisher(conn *amqp.Connection, exchangeName string, logger *slog.Logger) (EventPublisher, error) {
	if conn == nil {
		return nil, fmt.Errorf("RabbitMQ connection cannot be nil")
	}
	if exchangeName == "" {
		return nil, fmt.Errorf("RabbitMQ exchange name cannot be empty")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}

	tempCh, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open temporary channel for exchange declaration: %w", err)
	}
	defer tempCh.Close()

	err = tempCh.ExchangeDeclare(
		exchangeName,
		amqp.ExchangeTopic,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange '%s': %w", exchangeName, err)
	}
	logger.Info("Ensured RabbitMQ exchange exists", "exchange", exchangeName, "type", amqp.ExchangeTopic)

	return &RabbitMQEventPublisher{
		conn:         conn,
		exchangeName: exchangeName,
		logger:       logger.With("component", "RabbitMQEventPublisher", "exchange", exchangeName),
	}, nil
}

func (p *RabbitMQEventPublisher) publish(ctx context.Context, routingKey string, payload interface{}) error {
	logCtx := p.logger.With(slog.String("routingKey", routingKey))

	channel, err := p.conn.Channel()
	if err != nil {
		logCtx.ErrorContext(ctx, "Failed to open RabbitMQ channel", slog.Any("error", err))
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer channel.Close()

	body, err := json.Marshal(payload)
	if err != nil {
		logCtx.ErrorContext(ctx, "Failed to marshal event payload to JSON", slog.Any("error", err))
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	logCtx.DebugContext(ctx, "Publishing message", "bodySize", len(body))

	err = channel.PublishWithContext(
		ctx,
		p.exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
			AppId:        publisherAppID,
		},
	)

	if err != nil {
		logCtx.ErrorContext(ctx, "Failed to publish message to RabbitMQ", slog.Any("error", err))
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logCtx.InfoContext(ctx, "Successfully published message")
	return nil
}
