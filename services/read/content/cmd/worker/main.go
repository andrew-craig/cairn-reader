package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/jobs"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/repository"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

func main() {
	// Load .env file in development (before logger initialization)
	_ = godotenv.Load() // Ignore error as .env file is optional

	// Initialize structured logger
	logger := logging.NewLogger(logging.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "text"),
		ServiceName: "content-service-worker",
	})
	logging.SetDefault(logger)

	// Load configuration from environment variables
	cfg := loadConfig()

	// Initialize database connection
	slog.Info("component initializing", slog.String("component", "database"))
	db, err := database.NewConnection(cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	slog.Info("component initialized", slog.String("component", "database"))

	// Initialize repositories
	contentRepo := repository.NewContentRepository(db.DB)

	// Initialize jobs
	cleanupJob := jobs.NewCleanupJob(contentRepo, logger, 0)

	// Create cron scheduler
	scheduler := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))))

	// Schedule cleanup job to run daily at 2 AM
	cleanupCron := getEnv("CLEANUP_CRON", "0 2 * * *")
	_, err = scheduler.AddFunc(cleanupCron, cleanupJob.Run)
	if err != nil {
		slog.Error("failed to schedule cleanup job", slog.Any("error", err))
		os.Exit(1)
	}

	// Start the scheduler
	scheduler.Start()
	slog.Info("worker started successfully",
		slog.String("cleanup_schedule", cleanupCron),
	)
	slog.Info("background jobs")
	slog.Info("- orphaned content cleanup (runs daily at 2 AM)")

	// Start health check HTTP server
	healthPort := getEnv("HEALTH_PORT", "8082")
	healthServer := setupHealthCheckServer(healthPort, db.DB, logger)

	go func() {
		slog.Info("health check server starting", slog.String("port", healthPort))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health check server failed", slog.Any("error", err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker")

	// Stop the scheduler gracefully
	ctx := scheduler.Stop()
	<-ctx.Done()

	// Shutdown health check server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("health check server shutdown failed", slog.Any("error", err))
	}

	slog.Info("worker exited gracefully")
}

// Config holds application configuration
type Config struct {
	DB database.Config
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	// Database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnvAsInt("DB_PORT", 5433)
	dbUser := getEnv("DB_USER", "cairn_content")
	dbPassword := getEnv("DB_PASSWORD", "cairn_content_pass")
	dbName := getEnv("DB_NAME", "cairn_content")
	dbSSLMode := getEnv("DB_SSL_MODE", "disable")

	// Connection pool settings (smaller for worker process)
	maxOpenConns := getEnvAsInt("DB_MAX_OPEN_CONNS", 5)
	maxIdleConns := getEnvAsInt("DB_MAX_IDLE_CONNS", 2)
	connMaxLifetime := getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	connMaxIdleTime := getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 2*time.Minute)
	statementTimeout := getEnvAsDuration("DB_STATEMENT_TIMEOUT", 30*time.Second)

	return Config{
		DB: database.Config{
			Host:             dbHost,
			Port:             dbPort,
			User:             dbUser,
			Password:         dbPassword,
			DBName:           dbName,
			SSLMode:          dbSSLMode,
			MaxOpenConns:     maxOpenConns,
			MaxIdleConns:     maxIdleConns,
			ConnMaxLifetime:  connMaxLifetime,
			ConnMaxIdleTime:  connMaxIdleTime,
			StatementTimeout: statementTimeout,
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

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: invalid integer value for %s, using default %d", key, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsDuration gets an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Warning: invalid duration value for %s, using default %v", key, defaultValue)
		return defaultValue
	}
	return value
}

// setupHealthCheckServer creates an HTTP server for health checks
func setupHealthCheckServer(port string, db *sql.DB, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()

	// Liveness probe - always returns 200 if the process is running
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "content-service-worker",
		})
	})

	// Readiness probe - checks database connection
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check database connection
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			logger.Error("health check failed: database ping error", slog.Any("error", err))
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "unavailable",
				"error":  "database connection failed",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "content-service-worker",
		})
	})

	return &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}
