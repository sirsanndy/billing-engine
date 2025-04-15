package logging

import (
	"billing-engine/internal/config"
	"log/slog"
	"os"
	"strings"

	"github.com/go-chi/traceid"
)

func NewLogger(cfg config.LoggerConfig) *slog.Logger {
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

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	handler = traceid.LogHandler(handler)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
