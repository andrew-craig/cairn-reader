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
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn/pkg/logging"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/api"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/config"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/database"
)

func main() {
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
		ServiceName: "rss-fetcher-service",
	})
	logging.SetDefault(logger)

	slog.Info("starting service",
		slog.String("port", cfg.Server.Port),
	)

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

	if err := database.RunMigrations(dbConfig, migrationsPath); err != nil{
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

	// Create router
	router := api.NewRouter(db)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
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

	// Give outstanding requests time to complete
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", slog.Any("error", err))
	}

	slog.Info("server exited gracefully")
}
