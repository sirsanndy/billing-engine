package main

import (
	_ "billing-engine/docs"
	"billing-engine/internal/api"
	"billing-engine/internal/batch"
	"billing-engine/internal/config"
	"billing-engine/internal/domain/customer"
	"billing-engine/internal/domain/loan"
	"billing-engine/internal/infrastructure/database/postgres"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	loanService, customerService, loanRepo := initializeServices(dbPool, logger)

	updateJob := batch.NewUpdateDelinquencyJob(loanRepo, loanService, customerService, logger)

	cronScheduler := startBatchJobs(cfg, logger, updateJob)
	router := api.SetupRouter(loanService, customerService, cfg, logger)

	srv, serverErrors, shutdownChan := startServer(cfg, router, logger)
	handleShutdown(srv, cronScheduler, shutdownChan, serverErrors, logger)
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

func initializeServices(dbPool *pgxpool.Pool, logger *slog.Logger) (loan.LoanService, customer.CustomerService, loan.Repository) {
	logger.Info("Initializing application components...")
	loanRepo := postgres.NewLoanRepository(dbPool, logger)
	customerRepo := postgres.NewCustomerRepository(dbPool, logger)
	customerService := customer.NewCustomerService(customerRepo, logger)
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

func handleShutdown(srv *http.Server, cronScheduler *cron.Cron, shutdownChan <-chan os.Signal, serverErrors <-chan error, logger *slog.Logger) {
	select {
	case sig := <-shutdownChan:
		logger.Info("Shutdown signal received.", "signal", sig.String())
	case err := <-serverErrors:
		if err != nil {
			logger.Error("Server failed to start or exited unexpectedly", "error", err)
			os.Exit(1)
		}
		logger.Info("Server goroutine finished.")
		return
	}

	logger.Info("Starting graceful shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", "error", err)
		if err := srv.Close(); err != nil {
			logger.Error("Forced server close failed", "error", err)
		}
	} else {
		logger.Info("Server gracefully stopped.")
	}

	<-serverErrors
	logger.Info("Application shut down complete.")
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
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch strings.ToLower(cfg.Encoding) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
