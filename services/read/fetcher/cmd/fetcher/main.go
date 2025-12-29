// Package main is the entry point for the RSS fetcher service.
// The fetcher service is responsible for:
// - Managing user feed subscriptions
// - Fetching and parsing RSS/Atom feeds
// - Extracting feed items and sending them to the content service
// - Scheduling periodic feed updates
// - Providing subscription management API
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn/pkg/logging"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/api"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/database"
)

func main() {
	// Initialize structured logger
	logger := logging.NewLogger(logging.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "text"),
		ServiceName: "rss-fetcher-service",
	})
	logging.SetDefault(logger)

	// Load configuration from environment variables
	cfg := loadConfig()

	slog.Info("starting service",
		slog.String("port", cfg.Port),
	)

	// Initialize database connection
	slog.Info("component initializing", slog.String("component", "database"))
	db, err := database.NewConnection(&cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("component initialized", slog.String("component", "database"))

	// Create router
	router := api.NewRouter(db)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("server listening",
			slog.String("addr", server.Addr),
		)
		slog.Info("service ready to accept requests")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", slog.Any("error", err))
	}

	slog.Info("server exited gracefully")
}

// Config holds the application configuration
type Config struct {
	Port string
	DB   database.Config
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		Port: getEnv("PORT", "8081"),
		DB: database.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "cairn_rss"),
			Password: getEnv("DB_PASSWORD", "cairn_rss_pass"),
			DBName:   getEnv("DB_NAME", "cairn_rss"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
