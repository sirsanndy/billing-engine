package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"notify-service/internal/config"
	event "notify-service/internal/event/customer"
	"notify-service/internal/infrastructure/database/postgres"
	"notify-service/internal/infrastructure/logging"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg, logger := initializeConfigAndLogger()
	ctx, cancel := setupSignalHandling()
	defer cancel()

	dbpool := setupDatabase(ctx, cfg, logger)
	defer closeDatabase(dbpool, logger)

	rabbitConn := setupRabbitMQ(cfg, logger)
	defer closeRabbitMQ(rabbitConn, logger)

	customerRepo := postgres.NewCustomerRepository(dbpool, logger)
	eventHandler := event.NewCustomerEventHandler(customerRepo, logger)

	logger.Info("Setting up Prometheus metrics endpoint", "path", "/metrics")
	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: ":8090"}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server", slog.Any("error", err))
			cancel()
		}
	}()

	consumer := setupConsumer(rabbitConn, cfg, eventHandler, logger)
	go startConsumer(ctx, consumer, logger)

	waitForShutdownSignal(ctx, consumer, logger)

	logger.Info("Shutting down HTTP server...")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down HTTP server", slog.Any("error", err))
	}
	logger.Info("HTTP server shut down gracefully.")
}

func initializeConfigAndLogger() (*config.Config, *slog.Logger) {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	logger := logging.NewLogger(cfg.Logger)
	logger.Info("Configuration loaded successfully")
	return cfg, logger
}

func setupSignalHandling() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()
	return ctx, cancel
}

func setupDatabase(ctx context.Context, cfg *config.Config, logger *slog.Logger) *pgxpool.Pool {
	dbpool, err := postgres.NewConnectionPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("Database connection established")
	return dbpool
}

func closeDatabase(dbpool *pgxpool.Pool, logger *slog.Logger) {
	logger.Info("Closing database connection pool...")
	dbpool.Close()
}

func setupRabbitMQ(cfg *config.Config, logger *slog.Logger) *amqp.Connection {
	rabbitConn, err := connectRabbitMQ(cfg.RabbitMQ, logger)
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}
	return rabbitConn
}

func closeRabbitMQ(rabbitConn *amqp.Connection, logger *slog.Logger) {
	logger.Info("Closing RabbitMQ connection...")
	if err := rabbitConn.Close(); err != nil {
		logger.Error("Error closing RabbitMQ connection", slog.Any("error", err))
	}
}

func setupConsumer(rabbitConn *amqp.Connection, cfg *config.Config, eventHandler *event.CustomerEventHandler, logger *slog.Logger) *event.Consumer {
	consumer, err := event.NewConsumer(
		rabbitConn,
		cfg.RabbitMQ.ExchangeName,
		cfg.RabbitMQ.QueueName,
		cfg.RabbitMQ.ConsumerTag,
		eventHandler.HandleDelivery,
		logger,
	)
	if err != nil {
		logger.Error("Failed to create RabbitMQ consumer", slog.Any("error", err))
		os.Exit(1)
	}
	return consumer
}

func startConsumer(ctx context.Context, consumer *event.Consumer, logger *slog.Logger) {
	if err := consumer.Start(ctx); err != nil {
		logger.Error("Failed to start RabbitMQ consumer", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("Consumer started successfully. Waiting for events or shutdown signal...")
}

func waitForShutdownSignal(ctx context.Context, consumer *event.Consumer, logger *slog.Logger) {
	<-ctx.Done()
	logger.Info("Shutdown signal received. Initiating graceful shutdown...")
	consumer.Stop()
	logger.Info("Notify Service shut down gracefully.")
}

func connectRabbitMQ(cfg config.RabbitMQConfig, logger *slog.Logger) (*amqp.Connection, error) {
	logger.Info("Connecting to RabbitMQ", "uri", cfg.Host)
	uri := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
	)

	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	logger.Info("RabbitMQ connection established.")

	go func() {
		errChan := conn.NotifyClose(make(chan *amqp.Error))
		err = <-errChan
		logger.Error("RabbitMQ connection closed unexpectedly", slog.Any("error", err))
	}()

	return conn, nil
}
