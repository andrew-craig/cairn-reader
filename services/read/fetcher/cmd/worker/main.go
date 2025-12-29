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

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/client"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/database"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/jobs"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/processor"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/scheduler"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/worker"
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
	db, err := database.NewConnection(&cfg.DB)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connection established successfully")

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

	// Create cron scheduler for cleanup jobs
	cronScheduler := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))))

	// Schedule outbox cleanup job to run daily at 3 AM
	outboxCleanupCron := getEnv("OUTBOX_CLEANUP_CRON", "0 3 * * *")
	_, err = cronScheduler.AddFunc(outboxCleanupCron, outboxCleanupJob.Run)
	if err != nil {
		logger.Fatal("Failed to schedule outbox cleanup job", zap.Error(err))
	}

	// Schedule feed items cleanup job to run daily at 4 AM
	feedItemsCleanupCron := getEnv("FEED_ITEMS_CLEANUP_CRON", "0 4 * * *")
	_, err = cronScheduler.AddFunc(feedItemsCleanupCron, feedItemsCleanupJob.Run)
	if err != nil {
		logger.Fatal("Failed to schedule feed items cleanup job", zap.Error(err))
	}

	// Start all background workers
	feedWorker.Start()
	pollScheduler.Start()
	tierManager.Start()
	contentExtractionJob.Start()
	outboxWorker.Start()
	cronScheduler.Start()

	logger.Info("Worker started successfully")
	logger.Info("Background jobs:")
	logger.Info("- Feed polling scheduler (tiered polling strategy)")
	logger.Info("- Tier manager (daily tier updates)")
	logger.Info("- Content extraction job (processes pending feed items)")
	logger.Info("- Outbox delivery worker (delivers content to Content Service)")
	logger.Info("- Outbox cleanup job",
		zap.String("schedule", outboxCleanupCron),
	)
	logger.Info("- Feed items cleanup job",
		zap.String("schedule", feedItemsCleanupCron),
	)

	// Start health check HTTP server
	healthPort := getEnv("HEALTH_PORT", "8083")
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
		logger.Error("Health check server shutdown failed", zap.Error(err))
	}

	logger.Info("Worker exited")
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
			BaseURL:    contentServiceURL,
			Timeout:    contentServiceTimeout,
			MaxRetries: contentServiceMaxRetries,
			RetryDelay: contentServiceRetryDelay,
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
