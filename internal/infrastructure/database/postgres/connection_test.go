package postgres

import (
	"billing-engine/internal/config"
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

}

func TestConfigurePool(t *testing.T) {
	t.Run("ValidURL", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			URL: "postgres://user:password@host:8080/dbname?sslmode=disable",
		}

		poolConfig, err := configurePool(cfg)
		require.NoError(t, err)
		require.NotNil(t, poolConfig)

		assert.Equal(t, int32(10), poolConfig.MaxConns)
		assert.Equal(t, 5*time.Minute, poolConfig.MaxConnIdleTime)
		assert.Equal(t, 1*time.Minute, poolConfig.HealthCheckPeriod)

		assert.Equal(t, "host", poolConfig.ConnConfig.Host)
		assert.Equal(t, "dbname", poolConfig.ConnConfig.Database)
		assert.Equal(t, "user", poolConfig.ConnConfig.User)
	})

	t.Run("InvalidURL", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			URL: "://invalid-url-format",
		}

		poolConfig, err := configurePool(cfg)
		require.Error(t, err)
		assert.Nil(t, poolConfig)
		assert.Contains(t, err.Error(), "failed to parse database config from URL")
	})

	t.Run("EmptyURL", func(t *testing.T) {

		cfg := config.DatabaseConfig{
			URL: "",
		}
		poolConfig, err := configurePool(cfg)
		require.Error(t, err)
		assert.Nil(t, poolConfig)
		assert.Contains(t, err.Error(), "database URL is empty in configuration")
	})
}

func TestNewConnectionPool(t *testing.T) {
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("EmptyURLConfig", func(t *testing.T) {
		cfg := config.DatabaseConfig{
			URL: "",
		}

		dbpool, err := NewConnectionPool(ctx, cfg, logger)

		require.Error(t, err)
		assert.Nil(t, dbpool)
		assert.EqualError(t, err, "database URL is empty in configuration")
	})

	t.Run("InvalidURLFormat", func(t *testing.T) {

		cfg := config.DatabaseConfig{
			URL: "://invalid-url",
		}

		dbpool, err := NewConnectionPool(ctx, cfg, logger)

		require.Error(t, err)
		assert.Nil(t, dbpool)

		assert.Contains(t, err.Error(), "failed to parse database config from URL")
	})
}
