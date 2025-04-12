package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Run("Load default config when no config file is present", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "8080")
		os.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/billing_db?sslmode=disable")
		defer os.Unsetenv("SERVER_PORT")
		defer os.Unsetenv("DATABASE_URL")

		cfg, err := LoadConfig(".")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
		assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)
		assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)

		assert.Equal(t, "postgres://user:password@localhost:5432/billing_db?sslmode=disable", cfg.Database.URL)

		assert.Equal(t, "info", cfg.Logger.Level)
		assert.Equal(t, "json", cfg.Logger.Encoding)

		assert.Equal(t, 9090, cfg.Metrics.Port)
		assert.Equal(t, "/metrics", cfg.Metrics.Path)

		assert.Equal(t, 50, cfg.Loan.TermWeeks)
		assert.Equal(t, "0.10", cfg.Loan.InterestRate)

		assert.Equal(t, "0 2 * * *", cfg.Batch.DelinquencyUpdateSchedule)
		assert.Equal(t, time.Duration(30), cfg.Batch.DelinquencyUpdateTimeout)
	})

	t.Run("Return error when config file is invalid", func(t *testing.T) {
		invalidConfigPath := "./invalid_config"
		os.WriteFile(invalidConfigPath, []byte("invalid_yaml: : :"), 0644)
		defer os.Remove(invalidConfigPath)

		_, err := LoadConfig("./invalid_config")
		assert.NoError(t, err)
	})
}
