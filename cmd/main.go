package main

import (
	_ "billing-engine/docs"
	"billing-engine/internal/api"
	"billing-engine/internal/config"
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

// @host localhost:8080
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, logger := initializeApp()

	dbPool := initializeDatabase(cfg, logger)
	defer closeDatabase(dbPool, logger)

	loanService := initializeServices(dbPool, logger)
	router := api.SetupRouter(loanService, cfg, logger)

	startServer(cfg, router, logger)
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

func initializeServices(dbPool *pgxpool.Pool, logger *slog.Logger) loan.LoanService {
	logger.Info("Initializing application components...")
	loanRepo := postgres.NewLoanRepository(dbPool, logger)
	return loan.NewLoanService(loanRepo, logger)
}

func startServer(cfg *config.Config, router http.Handler, logger *slog.Logger) {
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

	handleShutdown(srv, shutdownChan, serverErrors, logger)
}

func handleShutdown(srv *http.Server, shutdownChan <-chan os.Signal, serverErrors <-chan error, logger *slog.Logger) {
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
