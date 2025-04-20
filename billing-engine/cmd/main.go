package main

import (
	_ "billing-engine/docs"
	"billing-engine/internal/api"
	"billing-engine/internal/api/middleware"
	"billing-engine/internal/batch"
	"billing-engine/internal/config"
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/event"
	"billing-engine/internal/infrastructure/database/postgres"
	"billing-engine/internal/infrastructure/logging"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
)

// @title Billing Engine API
// @version 1.0
// @description This is the API documentation for the Billing Engine service.
// @termsOfService http://billing-engine.com/terms/

// @contact.name API Support
// @contact.url http://billing-engine.com/support
// @contact.email support@billing-engine.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, logger := initializeApp()

	dbPool := initializeDatabase(cfg, logger)
	defer closeDatabase(dbPool, logger)
	rabbitMQConn, _ := setupRabbitMQ(cfg, logger)
	redisClient := initializeRedisClient(cfg, logger)
	rateLimiter := initializeRateLimiter(cfg, redisClient, logger)
	loanService, customerService, loanRepo := initializeServices(rabbitMQConn, dbPool, logger)

	updateJob := batch.NewUpdateDelinquencyJob(loanRepo, loanService, customerService, logger)

	cronScheduler := startBatchJobs(cfg, logger, updateJob)
	router := api.SetupRouter(rateLimiter, loanService, customerService, cfg, logger)

	srv, serverErrors, shutdownChan := startServer(cfg, router, logger)
	handleShutdown(srv, cronScheduler, rabbitMQConn, redisClient, shutdownChan, serverErrors, logger)
}

func initializeApp() (*config.Config, *slog.Logger) {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg.Logger)
	slog.SetDefault(logger)
	logger.Info("Application starting...", "config_source", viper.ConfigFileUsed())

	return cfg, logger
}

func initializeDatabase(cfg *config.Config, logger *slog.Logger) *pgxpool.Pool {
	logger.Info("Initializing database connection pool...")
	dbPool, err := postgres.NewConnectionPool(context.Background(), cfg.Database, logger)
	if err != nil {
		logger.Error("Failed to initialize database connection pool", "error", err)
		os.Exit(1)
	}
	return dbPool
}

func closeDatabase(dbPool *pgxpool.Pool, logger *slog.Logger) {
	logger.Info("Closing database connection pool...")
	dbPool.Close()
}

func initializeRateLimiter(cfg *config.Config, redisClient *redis.Client, logger *slog.Logger) *middleware.RateLimiterMiddleware {
	rlMiddleware := middleware.NewRateLimiterMiddleware(
		cfg.Server.RateLimit,
		redisClient,
		logger,
	)
	return rlMiddleware
}

func initializeServices(rabbitConn *amqp.Connection, dbPool *pgxpool.Pool, logger *slog.Logger) (loan.LoanService, customer.CustomerService, loan.Repository) {
	logger.Info("Initializing application components...")
	loanRepo := postgres.NewLoanRepository(dbPool, logger)
	customerRepo := postgres.NewCustomerRepository(dbPool, logger)
	eventPublisher, _ := event.NewRabbitMQEventPublisher(rabbitConn, "billing-engine", logger)
	customerService := customer.NewCustomerService(customerRepo, eventPublisher, logger)
	return loan.NewLoanService(loanRepo, customerService, logger), customerService, loanRepo
}

func startServer(cfg *config.Config, router http.Handler, logger *slog.Logger) (*http.Server, <-chan error, <-chan os.Signal) {
	logger.Info("Setting up HTTP server...", "port", cfg.Server.Port)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(fmt.Sprintf("Server listening on port %d", cfg.Server.Port))
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", "error", err)
			serverErrors <- err
		} else {
			logger.Info("Server closed gracefully.")
			serverErrors <- nil
		}
	}()
	return srv, serverErrors, shutdownChan
}

func handleShutdown(srv *http.Server, cronScheduler *cron.Cron, rabbitConn *amqp.Connection, redisClient *redis.Client,
	shutdownChan <-chan os.Signal, serverErrors <-chan error, logger *slog.Logger) {
	logger.Info("Shutdown handler started. Waiting for signal or server error...")

	triggerReason := waitForShutdownTrigger(shutdownChan, serverErrors, logger)

	logger.Info("Starting graceful shutdown...", "trigger", triggerReason)

	stopCronScheduler(cronScheduler, logger)
	closeRabbitMQConnection(rabbitConn, logger)
	closeRedisClient(redisClient, logger)
	shutdownHTTPServer(srv, serverErrors, logger)

	logger.Info("Application shutdown process complete.")
}

func waitForShutdownTrigger(shutdownChan <-chan os.Signal, serverErrors <-chan error, logger *slog.Logger) string {
	select {
	case sig := <-shutdownChan:
		logger.Info("Shutdown signal received.", "signal", sig.String())
		return "signal: " + sig.String()
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server exited unexpectedly before signal", "error", err)
			os.Exit(1)
		}
		logger.Info("Server goroutine finished before signal.", "error", err)
		return "server exited"
	}
}

func stopCronScheduler(cronScheduler *cron.Cron, logger *slog.Logger) {
	logger.Info("Stopping cron scheduler...")
	cronCtx := cronScheduler.Stop()
	select {
	case <-cronCtx.Done():
		logger.Info("Cron scheduler stopped gracefully.")
	case <-time.After(15 * time.Second):
		logger.Warn("Cron scheduler shutdown timed out.")
	}
}

func closeRabbitMQConnection(rabbitConn *amqp.Connection, logger *slog.Logger) {
	if rabbitConn != nil && !rabbitConn.IsClosed() {
		logger.Info("Closing RabbitMQ connection...")
		if err := rabbitConn.Close(); err != nil {
			logger.Error("Failed to close RabbitMQ connection gracefully", slog.Any("error", err))
		} else {
			logger.Info("RabbitMQ connection closed.")
		}
	} else if rabbitConn == nil {
		logger.Info("RabbitMQ connection was not established, skipping close.")
	} else {
		logger.Info("RabbitMQ connection already closed, skipping close.")
	}
}

func shutdownHTTPServer(srv *http.Server, serverErrors <-chan error, logger *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	logger.Info("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server graceful shutdown failed", "error", err)
		} else {
			logger.Info("HTTP server shutdown initiated.")
		}
		if err := srv.Close(); err != nil {
			logger.Error("HTTP server forced close failed", "error", err)
		}
	} else {
		logger.Info("HTTP server gracefully stopped.")
	}

	logger.Info("Waiting for server goroutine to confirm exit...")
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn("Server goroutine exited with unexpected error after shutdown", "error", err)
		} else {
			logger.Info("Server goroutine confirmed exit.")
		}
	case <-time.After(5 * time.Second):
		logger.Warn("Timed out waiting for server goroutine confirmation.")
	}
}

func initializeRedisClient(cfg *config.Config, logger *slog.Logger) *redis.Client {
	logger.Info("Initializing central Redis client...")
	if cfg.Redis.Addr == "" {
		logger.Error("Redis address (addr) is not configured.")
		os.Exit(1)
		return nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if status := rdb.Ping(ctx); status.Err() != nil {
		logger.Error("Failed to connect to Redis", "error", status.Err(), "addr", cfg.Redis.Addr)
		_ = rdb.Close()
		os.Exit(1)
		return nil
	}

	logger.Info("Central Redis client connected successfully.", "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)
	return rdb
}

func closeRedisClient(redisClient *redis.Client, logger *slog.Logger) {
	if redisClient != nil {
		logger.Info("Closing central Redis client connection...")
		if err := redisClient.Close(); err != nil {
			logger.Error("Failed to close central Redis client connection gracefully", "error", err)
		} else {
			logger.Info("Central Redis client connection closed.")
		}
	} else {
		logger.Info("Redis client was not initialized, skipping close.")
	}
}

func startBatchJobs(cfg *config.Config, logger *slog.Logger, updateJob *batch.UpdateDelinquencyJob) *cron.Cron {
	logger.Info("Initializing batch job scheduler...")
	c := cron.New()

	scheduleSpec := cfg.Batch.DelinquencyUpdateSchedule
	if scheduleSpec == "" {
		scheduleSpec = "0 2 * * *"
		logger.Warn("Batch delinquency update schedule not configured, using default", "schedule", scheduleSpec)
	}
	jobTimeout := cfg.Batch.DelinquencyUpdateTimeout
	if jobTimeout <= 0 {
		jobTimeout = 1 * time.Hour
	} else {
		jobTimeout = jobTimeout * time.Second
	}

	jobID, err := c.AddJob(scheduleSpec, cron.FuncJob(func() {
		jobLogger := logger.With("job_name", "DelinquencyUpdate")
		jobLogger.Info("Cron triggered: Running delinquency update job.")

		ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
		defer cancel()

		if runErr := updateJob.Run(ctx); runErr != nil {
			jobLogger.Error("Delinquency update job finished with error", slog.Any("error", runErr))
		} else {
			jobLogger.Info("Delinquency update job finished successfully.")
		}
	}))

	if err != nil {
		logger.Error("Failed to schedule delinquency update job", "schedule", scheduleSpec, slog.Any("error", err))

	} else {
		logger.Info("Scheduled delinquency update job", "schedule", scheduleSpec, "job_id", jobID)
	}

	c.Start()
	logger.Info("Cron scheduler started.")
	return c
}

func setupLogger(cfg config.LoggerConfig) *slog.Logger {
	return logging.NewLogger(cfg)
}

func connectRabbitMQ(uri string, logger *slog.Logger) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error
	retryCount := 5
	for i := 1; i <= retryCount; i++ {
		conn, err = amqp.Dial(uri)
		if err == nil {
			logger.Info("Successfully connected to RabbitMQ")

			go func() {
				blockChan := conn.NotifyBlocked(make(chan amqp.Blocking))
				closeChan := conn.NotifyClose(make(chan *amqp.Error))

				select {
				case b := <-blockChan:
					logger.Warn("RabbitMQ Connection Blocked", "reason", b.Reason)
				case e := <-closeChan:
					logger.Error("RabbitMQ Connection Closed", slog.Any("error", e))
				}
			}()

			return conn, nil
		}
		logger.Warn("Failed to connect to RabbitMQ, retrying...",
			slog.Int("attempt", i),
			slog.Int("max_attempts", retryCount),
			slog.Any("error", err),
		)
		time.Sleep(time.Duration(i*2) * time.Second)
	}
	return nil, fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", retryCount, err)
}

func setupRabbitMQ(cfg *config.Config, logger *slog.Logger) (*amqp.Connection, error) {
	rabbitMQURI := cfg.RabbitMQ.Host

	if rabbitMQURI == "" {
		return nil, fmt.Errorf("RabbitMQ host is not configured")
	}

	if cfg.RabbitMQ.Port != 0 {
		rabbitMQURI = fmt.Sprintf("amqp://%s:%d", cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	}

	if cfg.RabbitMQ.Username != "" && cfg.RabbitMQ.Password != "" {
		rabbitMQURI = fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.RabbitMQ.Username, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	} else if cfg.RabbitMQ.Username != "" || cfg.RabbitMQ.Password != "" {
		return nil, fmt.Errorf("RabbitMQ username and password must be provided together")
	}

	if rabbitMQURI == "" {
		return nil, fmt.Errorf("RabbitMQ URI is not configured")
	}

	conn, err := connectRabbitMQ(rabbitMQURI, logger)
	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", "error", err)
		return nil, err
	}
	return conn, nil
}
