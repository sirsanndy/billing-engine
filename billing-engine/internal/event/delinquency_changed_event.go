package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (p *RabbitMQEventPublisher) PublishCustomerDelinquencyChanged(ctx context.Context, event CustomerDelinquencyChangedEvent) error {
	logCtx := p.logger.With(
		slog.Int64("customerId", event.CustomerID),
		slog.Bool("newStatus", event.NewStatus),
	)

	channel, err := p.conn.Channel()
	if err != nil {
		logCtx.ErrorContext(ctx, "Failed to open RabbitMQ channel", slog.Any("error", err))
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer channel.Close()

	routingKey := "customer.delinquency.changed"

	body, err := json.Marshal(event)
	if err != nil {
		logCtx.ErrorContext(ctx, "Failed to marshal event payload to JSON", slog.Any("error", err))
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	logCtx.DebugContext(ctx, "Publishing message", "routingKey", routingKey, "bodySize", len(body))

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
			AppId:        "billing-engine",
		},
	)

	if err != nil {
		logCtx.ErrorContext(ctx, "Failed to publish message to RabbitMQ",
			slog.String("routingKey", routingKey),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logCtx.InfoContext(ctx, "Successfully published message", "routingKey", routingKey)
	return nil
}
