package postgres

import (
	"billing-engine/internal/config"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPgxPool struct {
	mock.Mock
}

func (m *MockPgxPool) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPgxPool) Close() {
	m.Called()
}

func TestNewConnectionPool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil))
	ctx := context.Background()

	t.Run("should return error when database URL is empty", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: ""}
		_, err := NewConnectionPool(ctx, cfg, logger)
		assert.Error(t, err)
		assert.Equal(t, "database URL is empty in configuration", err.Error())
	})

	t.Run("should return error when configurePool fails", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: "invalid-url"}
		_, err := NewConnectionPool(ctx, cfg, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse database config from URL")
	})

	t.Run("should return error when connection pool creation fails", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: "postgres://user:password@localhost:5432/dbname"}
		originalNewWithConfig := pgxpool.NewWithConfig
		defer func() { pgxpool.NewWithConfig = originalNewWithConfig }()
		pgxpool.NewWithConfig = func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
			return nil, errors.New("mock connection pool creation error")
		}

		_, err := NewConnectionPool(ctx, cfg, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to create connection pool")
	})

	t.Run("should return error when verifyConnection fails", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: "postgres://user:password@localhost:5432/dbname"}
		mockPool := new(MockPgxPool)
		mockPool.On("Ping", mock.Anything).Return(errors.New("mock ping error"))
		mockPool.On("Close").Return()

		originalNewWithConfig := pgxpool.NewWithConfig
		defer func() { pgxpool.NewWithConfig = originalNewWithConfig }()
		pgxpool.NewWithConfig = func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
			return mockPool, nil
		}

		_, err := NewConnectionPool(ctx, cfg, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database on connect")
		mockPool.AssertCalled(t, "Close")
	})

	t.Run("should successfully create connection pool", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: "postgres://user:password@localhost:5432/dbname"}
		mockPool := new(MockPgxPool)
		mockPool.On("Ping", mock.Anything).Return(nil)

		originalNewWithConfig := pgxpool.NewWithConfig
		defer func() { pgxpool.NewWithConfig = originalNewWithConfig }()
		pgxpool.NewWithConfig = func(ctx context.Context, config *pgxpool.Config) (*pgxpool.Pool, error) {
			return mockPool, nil
		}

		pool, err := NewConnectionPool(ctx, cfg, logger)
		assert.NoError(t, err)
		assert.NotNil(t, pool)
		mockPool.AssertCalled(t, "Ping", mock.Anything)
	})
}

func TestConfigurePool(t *testing.T) {
	t.Run("should return error for invalid database URL", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: "invalid-url"}
		_, err := configurePool(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse database config from URL")
	})

	t.Run("should configure pool successfully", func(t *testing.T) {
		cfg := config.DatabaseConfig{URL: "postgres://user:password@localhost:5432/dbname"}
		poolConfig, err := configurePool(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, poolConfig)
		assert.Equal(t, int32(10), poolConfig.MaxConns)
		assert.Equal(t, 5*time.Minute, poolConfig.MaxConnIdleTime)
		assert.Equal(t, 1*time.Minute, poolConfig.HealthCheckPeriod)
	})
}

func TestVerifyConnection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil))
	ctx := context.Background()

	t.Run("should return error when ping fails", func(t *testing.T) {
		mockPool := new(MockPgxPool)
		mockPool.On("Ping", mock.Anything).Return(errors.New("mock ping error"))

		err := verifyConnection(ctx, mockPool, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to ping database on connect")
		mockPool.AssertCalled(t, "Ping", mock.Anything)
	})

	t.Run("should verify connection successfully", func(t *testing.T) {
		mockPool := new(MockPgxPool)
		mockPool.On("Ping", mock.Anything).Return(nil)

		err := verifyConnection(ctx, mockPool, logger)
		assert.NoError(t, err)
		mockPool.AssertCalled(t, "Ping", mock.Anything)
	})
}
