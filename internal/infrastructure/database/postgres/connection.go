package postgres

import (
	"billing-engine/internal/config"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnectionPool(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (*pgxpool.Pool, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database URL is empty in configuration")
	}

	poolConfig, err := configurePool(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Connecting to PostgreSQL database...")
	dbpool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := verifyConnection(ctx, dbpool, logger); err != nil {
		dbpool.Close()
		return nil, err
	}

	logger.Info("Successfully connected to PostgreSQL database.", "host", poolConfig.ConnConfig.Host, "db", poolConfig.ConnConfig.Database)
	return dbpool, nil
}

func configurePool(cfg config.DatabaseConfig) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config from URL: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	return poolConfig, nil
}

func verifyConnection(ctx context.Context, dbpool *pgxpool.Pool, logger *slog.Logger) error {
	logger.Info("Pinging database...")
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := dbpool.Ping(pingCtx); err != nil {
		logger.Error("Failed to ping database", "error", err)
		return fmt.Errorf("failed to ping database on connect: %w", err)
	}

	return nil
}
