package event

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	routingKeyCustomerCreated = "customer.created"
	routingKeyCustomerUpdated = "customer.updated"
)

type MessageHandler func(ctx context.Context, d amqp.Delivery)

type Consumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	queueName    string
	consumerTag  string
	handler      MessageHandler
	logger       *slog.Logger
	wg           *sync.WaitGroup
	cancelFunc   context.CancelFunc
}

func NewConsumer(
	conn *amqp.Connection,
	exchangeName, queueName, consumerTag string,
	handler MessageHandler,
	logger *slog.Logger,
) (*Consumer, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	logger.Info("Declaring exchange", "name", exchangeName, "type", amqp.ExchangeTopic)
	err = ch.ExchangeDeclare(exchangeName, amqp.ExchangeTopic, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("failed to declare exchange '%s': %w", exchangeName, err)
	}

	logger.Info("Declaring queue", "name", queueName)
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("failed to declare queue '%s': %w", queueName, err)
	}

	routingKeys := []string{routingKeyCustomerCreated, routingKeyCustomerUpdated}
	for _, key := range routingKeys {
		logger.Info("Binding queue", "queue", q.Name, "exchange", exchangeName, "key", key)
		err = ch.QueueBind(q.Name, key, exchangeName, false, nil)
		if err != nil {
			_ = ch.Close()
			return nil, fmt.Errorf("failed to bind queue '%s' with key '%s': %w", q.Name, key, err)
		}
	}

	prefetchCount := 1
	err = ch.Qos(prefetchCount, 0, false)
	if err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &Consumer{
		conn:         conn,
		channel:      ch,
		exchangeName: exchangeName,
		queueName:    q.Name,
		consumerTag:  consumerTag,
		handler:      handler,
		logger:       logger.With("component", "consumer", "queue", q.Name),
		wg:           new(sync.WaitGroup),
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting message consumption...")
	deliveries, err := c.channel.Consume(
		c.queueName,
		c.consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = c.channel.Close()
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	loopCtx, cancel := context.WithCancel(ctx)
	c.cancelFunc = cancel

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.logger.Info("Consumer goroutine started.")
		for {
			select {
			case <-loopCtx.Done():
				c.logger.Info("Consumer context cancelled. Exiting consumption loop.")
				return
			case d, ok := <-deliveries:
				if !ok {
					c.logger.Warn("RabbitMQ delivery channel closed unexpectedly.")

					return
				}

				c.handler(loopCtx, d)
			}
		}
	}()

	return nil
}

func (c *Consumer) Stop() {
	if c.cancelFunc == nil {
		c.logger.Warn("Consumer stop called but cancelFunc is nil (maybe never started?)")
		return
	}
	c.logger.Info("Stopping consumer...")

	c.cancelFunc()

	if err := c.channel.Cancel(c.consumerTag, false); err != nil {
		c.logger.Warn("Failed to cancel consumer tag", "tag", c.consumerTag, "error", err)
	}

	c.logger.Info("Waiting for consumer goroutine to exit...")
	c.wg.Wait()
	c.logger.Info("Consumer goroutine finished.")

	if err := c.channel.Close(); err != nil {
		c.logger.Error("Failed to close consumer channel", "error", err)
	} else {
		c.logger.Info("Consumer channel closed.")
	}
}
