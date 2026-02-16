// Package main is the entry point for the Email Ingest Worker.
// The worker is responsible for:
// - Processing raw emails from the database
// - Email-specific cleaning (tracking pixels, footers)
// - Content extraction using readability
// - Delivering processed content to the outbox
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/config"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file in development (before logger initialization)
	_ = godotenv.Load() // Ignore error as .env file is optional

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize structured logger
	logger := logging.NewLogger(logging.Config{
		Level:       cfg.Logging.Level,
		Format:      cfg.Logging.Format,
		ServiceName: "email-ingest-worker",
	})
	logging.SetDefault(logger)

	slog.Info("starting worker")

	// Run database migrations
	slog.Info("running database migrations")
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		slog.Error("failed to get migrations path", slog.Any("error", err))
		os.Exit(1)
	}

	// Convert database config for migrations
	port, _ := strconv.Atoi(cfg.Database.Port)
	dbConfig := &database.Config{
		Host:     cfg.Database.Host,
		Port:     port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	if err := database.RunMigrations(dbConfig, migrationsPath); err != nil {
		slog.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	// Initialize database connection
	slog.Info("component initializing", slog.String("component", "database"))
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("component initialized", slog.String("component", "database"))

	// TODO: Initialize workers
	// - Email processor worker
	// - Outbox delivery worker
	// - Cleanup jobs

	slog.Info("worker ready")

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker")

	// TODO: Gracefully stop all workers with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	_ = ctx

	slog.Info("worker exited gracefully")
}
