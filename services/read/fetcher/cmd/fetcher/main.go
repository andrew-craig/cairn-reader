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
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn-read/fetcher/internal/api"
	"github.com/andrew-craig/cairn-read/fetcher/internal/database"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration from environment variables
	cfg := loadConfig()

	logger.Info("Starting RSS fetcher service",
		zap.String("port", cfg.Port),
	)

	// Initialize database connection
	db, err := database.NewConnection(&cfg.DB)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connection established successfully")

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
		logger.Info("Server started",
			zap.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
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
