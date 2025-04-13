package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"notify-service/internal/domain/customer"
	"notify-service/internal/infrastructure/monitoring"

	amqp "github.com/rabbitmq/amqp091-go"
)

type CustomerEventHandler struct {
	repo   customer.CustomerRepository
	logger *slog.Logger
}

func NewCustomerEventHandler(repo customer.CustomerRepository, logger *slog.Logger) *CustomerEventHandler {
	return &CustomerEventHandler{
		repo:   repo,
		logger: logger.With("component", "CustomerEventHandler"),
	}
}

func (h *CustomerEventHandler) HandleDelivery(ctx context.Context, d amqp.Delivery) {
	logCtx := h.logger.With(slog.Uint64("deliveryTag", d.DeliveryTag), slog.String("routingKey", d.RoutingKey))
	processed := false

	defer func() {
		if !processed {
			logCtx.WarnContext(ctx, "Message processing ended without explicit Ack/Nack")
			_ = d.Nack(false, false)
		}
	}()

	var payload CustomerEventPayload

	switch d.RoutingKey {
	case routingKeyCustomerCreated:
		var event CustomerCreatedEvent
		if err := json.Unmarshal(d.Body, &event); err != nil {
			logCtx.ErrorContext(ctx, "Failed to unmarshal CustomerCreatedEvent", "error", err, "body", string(d.Body))
			_ = d.Nack(false, false)
			processed = true
			return
		}
		payload = event.Payload
	case routingKeyCustomerUpdated:
		var event CustomerUpdatedEvent
		if err := json.Unmarshal(d.Body, &event); err != nil {
			logCtx.ErrorContext(ctx, "Failed to unmarshal CustomerUpdatedEvent", "error", err, "body", string(d.Body))
			_ = d.Nack(false, false)
			processed = true
			return
		}
		payload = event.Payload
	default:
		logCtx.WarnContext(ctx, "Received message with unknown routing key. Discarding.")
		_ = d.Reject(false)
		processed = true
		return
	}

	var customerToUpsert *customer.Customer = &customer.Customer{
		CustomerID:   payload.CustomerID,
		Name:         payload.Name,
		Address:      payload.Address,
		IsDelinquent: payload.IsDelinquent,
		Active:       payload.Active,
		LoanID:       payload.LoanID,
		CreatedAt:    payload.CreateDate,
		UpdatedAt:    payload.UpdatedAt,
	}

	logCtx = logCtx.With(slog.Int64("customerID", customerToUpsert.CustomerID))
	logCtx.InfoContext(ctx, "Processing event for customer")
	monitoring.RecordConsumerProcessed()
	if err := h.repo.Upsert(ctx, customerToUpsert); err != nil {
		logCtx.ErrorContext(ctx, "Failed to upsert customer via repository", "error", err)

		_ = d.Nack(false, false)
		processed = true
		return
	}

	if err := d.Ack(false); err != nil {
		logCtx.ErrorContext(ctx, "Failed to acknowledge message after successful processing", "error", err)

	} else {
		logCtx.InfoContext(ctx, "Successfully processed and acknowledged message")
	}
	processed = true
}
