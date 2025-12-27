// Package logging provides structured logging using the standard library log/slog package.
package logging

import (
	"log/slog"
	"os"
)

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Format      string // json, text
	ServiceName string // e.g., "fetcher", "recommender"
}

// NewLogger creates a configured slog.Logger
func NewLogger(cfg Config) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		// Add source file info for debug/error levels
		AddSource: level <= slog.LevelDebug || level >= slog.LevelError,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Add service name as a default attribute
	return slog.New(handler).With(
		slog.String("service", cfg.ServiceName),
	)
}

// SetDefault sets the default logger for the application
func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}
