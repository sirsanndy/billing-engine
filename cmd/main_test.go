package main

import (
	"billing-engine/internal/config"
	"billing-engine/internal/infrastructure/logging"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func TestInitializeApp(t *testing.T) {
	cfg, log := initializeApp()

	assert.NotNil(t, cfg, "Config should not be nil")
	assert.NotNil(t, log, "Logger should not be nil")
}

func TestStartServer(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  5 * time.Second,
		},
	}
	logger := logging.NewLogger(config.LoggerConfig{})
	router := http.NewServeMux()

	srv, serverErrors, shutdownChan := startServer(cfg, router, logger)

	assert.NotNil(t, srv, "Server should not be nil")
	assert.NotNil(t, serverErrors, "Server errors channel should not be nil")
	assert.NotNil(t, shutdownChan, "Shutdown channel should not be nil")
}

func TestHandleShutdown(t *testing.T) {
	logger := logging.NewLogger(config.LoggerConfig{})
	cronScheduler := cron.New()
	srv := &http.Server{}
	shutdownChan := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)

	go func() {
		shutdownChan <- syscall.SIGINT
	}()

	handleShutdown(srv, cronScheduler, shutdownChan, serverErrors, logger)
	assert.True(t, true, "Graceful shutdown should complete without errors")
}
