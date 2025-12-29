package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/database"
	"github.com/andrew-craig/cairn/services/read/content/internal/jobs"
	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
	"github.com/robfig/cron/v3"
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

	// Initialize database connection
	db, err := database.NewConnection(cfg.DB)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connection established successfully")

	// Initialize repositories
	contentRepo := repository.NewContentRepository(db.DB)

	// Initialize jobs
	cleanupJob := jobs.NewCleanupJob(contentRepo, logger)

	// Create cron scheduler
	scheduler := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))))

	// Schedule cleanup job to run daily at 2 AM
	cleanupCron := getEnv("CLEANUP_CRON", "0 2 * * *")
	_, err = scheduler.AddFunc(cleanupCron, cleanupJob.Run)
	if err != nil {
		logger.Fatal("Failed to schedule cleanup job", zap.Error(err))
	}

	// Start the scheduler
	scheduler.Start()
	logger.Info("Worker started successfully",
		zap.String("cleanup_schedule", cleanupCron),
	)
	logger.Info("Background jobs:")
	logger.Info("- Orphaned content cleanup (runs daily at 2 AM)")

	// Start health check HTTP server
	healthPort := getEnv("HEALTH_PORT", "8082")
	healthServer := setupHealthCheckServer(healthPort, db.DB, logger)

	go func() {
		logger.Info("Health check server starting", zap.String("port", healthPort))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health check server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")

	// Stop the scheduler gracefully
	ctx := scheduler.Stop()
	<-ctx.Done()

	// Shutdown health check server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Health check server shutdown failed", zap.Error(err))
	}

	logger.Info("Worker exited")
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

	return Config{
		DB: database.Config{
			Host:            dbHost,
			Port:            dbPort,
			User:            dbUser,
			Password:        dbPassword,
			DBName:          dbName,
			SSLMode:         dbSSLMode,
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
			ConnMaxIdleTime: connMaxIdleTime,
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
func setupHealthCheckServer(port string, db *sql.DB, logger *zap.Logger) *http.Server {
	mux := http.NewServeMux()

	// Liveness probe - always returns 200 if the process is running
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
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
			logger.Error("Health check failed: database ping error", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unavailable",
				"error":  "database connection failed",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
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
