package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/jobs"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/processor"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/scheduler"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/worker"
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
		ServiceName: "rss-fetcher-service-worker",
	})
	logging.SetDefault(logger)

	// Load configuration from environment variables
	cfg := loadConfig()

	// Initialize database connection
	slog.Info("component initializing", slog.String("component", "database"))
	db, err := database.NewConnection(&cfg.DB)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	slog.Info("component initialized", slog.String("component", "database"))

	// Initialize repositories
	feedRepo := repository.NewFeedRepository(db.DB)
	subscriptionRepo := repository.NewFeedSubscriptionRepository(db.DB)
	feedItemRepo := repository.NewFeedItemRepository(db.DB)
	outboxRepo := repository.NewOutboxRepository(db.DB)

	// Initialize Content Service client
	contentServiceClient := client.NewContentServiceClient(cfg.ContentService)

	// Initialize feed worker pool
	feedWorkerConfig := &worker.FeedWorkerConfig{
		WorkerCount: cfg.FeedWorker.WorkerCount,
		QueueSize:   cfg.FeedWorker.QueueSize,
	}
	noOpProcessor := &worker.NoOpFeedProcessor{}
	feedWorker := worker.NewFeedWorker(feedWorkerConfig, feedRepo, noOpProcessor)

	// Initialize poll scheduler
	pollScheduler := scheduler.NewPollScheduler(cfg.PollScheduler, feedRepo, feedWorker)

	// Initialize tier manager
	tierManager := scheduler.NewTierManager(cfg.TierManager, feedRepo)

	// Initialize item processor
	itemProcessor := processor.NewItemProcessor(cfg.ItemProcessor, feedItemRepo, subscriptionRepo, outboxRepo)

	// Initialize content extraction job
	contentExtractionJob := jobs.NewContentExtractionJob(cfg.ContentExtraction, itemProcessor)

	// Initialize outbox worker
	outboxWorker := worker.NewOutboxWorker(cfg.OutboxWorker, outboxRepo, contentServiceClient)

	// Initialize cleanup jobs
	outboxCleanupJob := jobs.NewOutboxCleanupJob(cfg.OutboxCleanup, outboxRepo)
	feedItemsCleanupJob := jobs.NewFeedItemsCleanupJob(cfg.FeedItemsCleanup, feedItemRepo)

	// Create cron scheduler for cleanup jobs (with default logger)
	cronScheduler := cron.New()

	// Schedule outbox cleanup job to run daily at 3 AM
	outboxCleanupCron := getEnv("OUTBOX_CLEANUP_CRON", "0 3 * * *")
	_, err = cronScheduler.AddFunc(outboxCleanupCron, outboxCleanupJob.Run)
	if err != nil {
		slog.Error("failed to schedule outbox cleanup job", slog.Any("error", err))
		os.Exit(1)
	}

	// Schedule feed items cleanup job to run daily at 4 AM
	feedItemsCleanupCron := getEnv("FEED_ITEMS_CLEANUP_CRON", "0 4 * * *")
	_, err = cronScheduler.AddFunc(feedItemsCleanupCron, feedItemsCleanupJob.Run)
	if err != nil {
		slog.Error("failed to schedule feed items cleanup job", slog.Any("error", err))
		os.Exit(1)
	}

	// Start all background workers
	feedWorker.Start()
	pollScheduler.Start()
	tierManager.Start()
	contentExtractionJob.Start()
	outboxWorker.Start()
	cronScheduler.Start()

	slog.Info("worker started successfully")
	slog.Info("background jobs")
	slog.Info("- feed polling scheduler (tiered polling strategy)")
	slog.Info("- tier manager (daily tier updates)")
	slog.Info("- content extraction job (processes pending feed items)")
	slog.Info("- outbox delivery worker (delivers content to Content Service)")
	slog.Info("- outbox cleanup job",
		slog.String("schedule", outboxCleanupCron),
	)
	slog.Info("- feed items cleanup job",
		slog.String("schedule", feedItemsCleanupCron),
	)

	// Start health check HTTP server
	healthPort := getEnv("HEALTH_PORT", "8083")
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

	// Stop all background workers in reverse order
	ctx := cronScheduler.Stop()
	<-ctx.Done()
	outboxWorker.Stop()
	contentExtractionJob.Stop()
	tierManager.Stop()
	pollScheduler.Stop()
	feedWorker.Stop()

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
	DB                database.Config
	ContentService    client.ContentServiceConfig
	FeedWorker        FeedWorkerConfig
	PollScheduler     *scheduler.PollSchedulerConfig
	TierManager       *scheduler.TierManagerConfig
	ItemProcessor     *processor.ItemProcessorConfig
	ContentExtraction *jobs.ContentExtractionJobConfig
	OutboxWorker      *worker.OutboxWorkerConfig
	OutboxCleanup     *jobs.OutboxCleanupJobConfig
	FeedItemsCleanup  *jobs.FeedItemsCleanupJobConfig
}

// FeedWorkerConfig holds feed worker configuration
type FeedWorkerConfig struct {
	WorkerCount int
	QueueSize   int
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	// Database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnvAsInt("DB_PORT", 5434)
	dbUser := getEnv("DB_USER", "cairn_rss")
	dbPassword := getEnv("DB_PASSWORD", "cairn_rss_pass")
	dbName := getEnv("DB_NAME", "cairn_rss_fetcher")
	dbSSLMode := getEnv("DB_SSL_MODE", "disable")

	// Connection pool settings (smaller for worker process)
	maxOpenConns := getEnvAsInt("DB_MAX_OPEN_CONNS", 10)
	maxIdleConns := getEnvAsInt("DB_MAX_IDLE_CONNS", 5)
	connMaxLifetime := getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)
	connMaxIdleTime := getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 2*time.Minute)

	// Content Service configuration
	contentServiceURL := getEnv("CONTENT_SERVICE_URL", "http://localhost:8080")
	contentServiceTimeout := getEnvAsDuration("CONTENT_SERVICE_TIMEOUT", 30*time.Second)
	contentServiceMaxRetries := getEnvAsInt("CONTENT_SERVICE_MAX_RETRIES", 3)
	contentServiceRetryDelay := getEnvAsDuration("CONTENT_SERVICE_RETRY_DELAY", 1*time.Second)
	internalAPIKey := getEnv("INTERNAL_API_KEY", "")

	// Feed worker configuration
	feedWorkerCount := getEnvAsInt("FEED_WORKER_COUNT", 5)
	feedWorkerQueueSize := getEnvAsInt("FEED_WORKER_QUEUE_SIZE", 100)

	// Poll scheduler configuration
	pollBatchSize := getEnvAsInt("POLL_BATCH_SIZE", 50)
	pollInterval := getEnvAsDuration("POLL_INTERVAL", 1*time.Minute)
	pollWorkerPoolSize := getEnvAsInt("POLL_WORKER_POOL_SIZE", 5)

	// Tier manager configuration
	tierUpdateInterval := getEnvAsDuration("TIER_UPDATE_INTERVAL", 24*time.Hour)

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
		ContentService: client.ContentServiceConfig{
			BaseURL:        contentServiceURL,
			Timeout:        contentServiceTimeout,
			MaxRetries:     contentServiceMaxRetries,
			RetryDelay:     contentServiceRetryDelay,
			InternalAPIKey: internalAPIKey,
		},
		FeedWorker: FeedWorkerConfig{
			WorkerCount: feedWorkerCount,
			QueueSize:   feedWorkerQueueSize,
		},
		PollScheduler: &scheduler.PollSchedulerConfig{
			BatchSize:      pollBatchSize,
			PollInterval:   pollInterval,
			WorkerPoolSize: pollWorkerPoolSize,
		},
		TierManager: &scheduler.TierManagerConfig{
			UpdateInterval: tierUpdateInterval,
		},
		ItemProcessor:     processor.DefaultItemProcessorConfig(),
		ContentExtraction: jobs.DefaultContentExtractionJobConfig(),
		OutboxWorker:      worker.DefaultOutboxWorkerConfig(),
		OutboxCleanup:     jobs.DefaultOutboxCleanupJobConfig(),
		FeedItemsCleanup:  jobs.DefaultFeedItemsCleanupJobConfig(),
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
		slog.Warn("invalid integer value for environment variable, using default",
			slog.String("key", key),
			slog.String("value", valueStr),
			slog.Int("default", defaultValue),
		)
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
		slog.Warn("invalid duration value for environment variable, using default",
			slog.String("key", key),
			slog.String("value", valueStr),
			slog.Duration("default", defaultValue),
		)
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
			"service": "rss-fetcher-service-worker",
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
			"service": "rss-fetcher-service-worker",
		})
	})

	return &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}
