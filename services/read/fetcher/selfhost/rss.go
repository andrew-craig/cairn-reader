// Package selfhost provides initialization functions for the consolidated self-hosted binary.
package selfhost

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"time"

	rssAPI "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/api"
	rssClient "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/client"
	rssDB "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/database"
	rssFetcher "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/fetcher"
	rssJobs "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/jobs"
	rssProcessor "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/processor"
	rssRepo "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	rssScheduler "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/scheduler"
	rssWorker "github.com/cairn-app/cairn-reader/services/read/fetcher/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/robfig/cron/v3"
)

// envInt returns the int value of the given env var, or fallback if unset/invalid.
func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// RSSConfig holds configuration for the ingest RSS service.
type RSSConfig struct {
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	ContentBaseURL string
	InternalAPIKey string
}

// RunMigrations runs embedded migrations for the RSS database.
func RunMigrations(cfg RSSConfig, migrations fs.FS) error {
	dbConfig := &rssDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
	return rssDB.RunMigrationsFS(dbConfig, migrations)
}

// Mount initializes the RSS ingest service and mounts routes.
// Returns the underlying *sql.DB for health checks, a cleanup function, and any error.
func Mount(cfg RSSConfig, r chi.Router, logger *slog.Logger) (*sql.DB, func(), error) {
	dbConfig := &rssDB.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		DBName:          cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    envInt("DB_MAX_CONNS", 3),
		MaxIdleConns:    envInt("DB_MIN_CONNS", 1),
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}

	db, err := rssDB.NewConnection(dbConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("rss db: %w", err)
	}

	rssRouter := rssAPI.NewRouter(db)
	r.Handle("/api/v1/source/rss", rssRouter)
	r.Handle("/api/v1/source/rss/*", rssRouter)

	feedRepo := rssRepo.NewFeedRepository(db.DB)
	subscriptionRepo := rssRepo.NewFeedSubscriptionRepository(db.DB)
	feedItemRepo := rssRepo.NewFeedItemRepository(db.DB)
	outboxRepo := rssRepo.NewOutboxRepository(db.DB)

	contentServiceClient := rssClient.NewContentServiceClient(rssClient.ContentServiceConfig{
		BaseURL:        cfg.ContentBaseURL,
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
		InternalAPIKey: cfg.InternalAPIKey,
	})

	feedFetcher := rssFetcher.NewFeedFetcher(rssFetcher.DefaultFeedFetcherConfig(), feedRepo, feedItemRepo)
	feedWorker := rssWorker.NewFeedWorker(&rssWorker.FeedWorkerConfig{
		WorkerCount: 5,
		QueueSize:   100,
	}, feedRepo, feedFetcher)

	pollScheduler := rssScheduler.NewPollScheduler(&rssScheduler.PollSchedulerConfig{
		BatchSize:      50,
		PollInterval:   1 * time.Minute,
		WorkerPoolSize: 5,
	}, feedRepo, feedWorker)

	tierManager := rssScheduler.NewTierManager(&rssScheduler.TierManagerConfig{
		UpdateInterval: 24 * time.Hour,
	}, feedRepo)

	itemProcessor := rssProcessor.NewItemProcessor(
		rssProcessor.DefaultItemProcessorConfig(),
		feedItemRepo, subscriptionRepo, outboxRepo,
	)

	contentExtractionJob := rssJobs.NewContentExtractionJob(
		rssJobs.DefaultContentExtractionJobConfig(), itemProcessor,
	)

	outboxWorker := rssWorker.NewOutboxWorker(
		rssWorker.DefaultOutboxWorkerConfig(), outboxRepo, feedItemRepo, contentServiceClient,
	)

	outboxCleanupJob := rssJobs.NewOutboxCleanupJob(
		rssJobs.DefaultOutboxCleanupJobConfig(), outboxRepo,
	)
	feedItemsCleanupJob := rssJobs.NewFeedItemsCleanupJob(
		rssJobs.DefaultFeedItemsCleanupJobConfig(), feedItemRepo,
	)

	cronScheduler := cron.New()
	_, _ = cronScheduler.AddFunc("0 3 * * *", outboxCleanupJob.Run)
	_, _ = cronScheduler.AddFunc("0 4 * * *", feedItemsCleanupJob.Run)

	feedWorker.Start()
	pollScheduler.Start()
	tierManager.Start()
	contentExtractionJob.Start()
	outboxWorker.Start()
	cronScheduler.Start()

	return db.DB, func() {
		cronCtx := cronScheduler.Stop()
		<-cronCtx.Done()
		outboxWorker.Stop()
		contentExtractionJob.Stop()
		tierManager.Stop()
		pollScheduler.Stop()
		feedWorker.Stop()
		_ = db.Close()
	}, nil
}
