package postgres

import (
	"billing-engine/internal/config" // Adjust import path if needed
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a discard logger for tests
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})) // Use Stderr for visibility during testing if needed, or io.DiscardHandler
	// return slog.New(slog.NewTextHandler(io.Discard, nil)) // Use io.Discard to suppress logs during tests
}

func TestConfigurePool(t *testing.T) {
	t.Run("ValidURL", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			URL: "postgres://user:password@host:port/dbname?sslmode=disable",
		}

		poolConfig, err := configurePool(cfg)
		require.NoError(t, err)
		require.NotNil(t, poolConfig)

		// Check default settings applied
		assert.Equal(t, int32(10), poolConfig.MaxConns)
		assert.Equal(t, 5*time.Minute, poolConfig.MaxConnIdleTime)
		assert.Equal(t, 1*time.Minute, poolConfig.HealthCheckPeriod)

		// Check basic parsing from URL
		assert.Equal(t, "host", poolConfig.ConnConfig.Host)
		assert.Equal(t, "dbname", poolConfig.ConnConfig.Database)
		assert.Equal(t, "user", poolConfig.ConnConfig.User)
	})

	t.Run("InvalidURL", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			URL: "://invalid-url-format", // Malformed URL
		}

		poolConfig, err := configurePool(cfg)
		require.Error(t, err)
		assert.Nil(t, poolConfig)
		assert.Contains(t, err.Error(), "failed to parse database config from URL")
	})

	t.Run("EmptyURL", func(t *testing.T) {
		// configurePool expects a non-empty URL as pgxpool.ParseConfig does
		cfg := config.DatabaseConfig{
			URL: "",
		}
		poolConfig, err := configurePool(cfg)
		require.Error(t, err) // pgxpool.ParseConfig returns error for empty string
		assert.Nil(t, poolConfig)
		assert.Contains(t, err.Error(), "cannot be blank") // Error message from pgxpool
	})
}

func TestNewConnectionPool(t *testing.T) {
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("EmptyURLConfig", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			URL: "", // Explicitly empty URL
		}

		dbpool, err := NewConnectionPool(ctx, cfg, logger)

		require.Error(t, err)
		assert.Nil(t, dbpool)
		assert.EqualError(t, err, "database URL is empty in configuration")
	})

	t.Run("InvalidURLFormat", func(t *testing.T) {
		// This tests the error propagation from configurePool
		cfg := config.DatabaseConfig{
			URL: "://invalid-url",
		}

		dbpool, err := NewConnectionPool(ctx, cfg, logger)

		require.Error(t, err)
		assert.Nil(t, dbpool)
		// We expect the error message from configurePool, wrapped by its caller if applicable (here it's not wrapped further)
		assert.Contains(t, err.Error(), "failed to parse database config from URL")
	})
}
