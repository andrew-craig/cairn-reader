// Package main is the entry point for the fetcher service.
// The fetcher service is responsible for:
// - Syncing feed sources from the Kagi Small Web collection (daily)
// - Fetching RSS feeds on a scheduled basis (1 feed per minute)
// - Submitting fetched articles to the recommender service via HTTP
// - Providing HTTP endpoints for health checks and manual triggers
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/pkg/logging"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/api"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/client"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/config"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/sync"
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
		ServiceName: "fetcher",
	})
	logging.SetDefault(logger)

	slog.Info("starting service",
		slog.String("port", cfg.Server.Port),
		slog.Duration("fetch_interval", cfg.FetchInterval),
	)

	// Run database migrations
	slog.Info("running database migrations")
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		slog.Error("failed to get migrations path", slog.Any("error", err))
		os.Exit(1)
	}

	if err := db.RunMigrations(migrationsPath); err != nil {
		slog.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	// Initialize database connection
	ctx := context.Background()
	slog.Info("connecting to database")
	dbConfig := &db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}
	database, err := dbConfig.Connect(ctx)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("component initialized", slog.String("component", "database"))

	// Initialize repositories
	feedRepo := db.NewFeedRepository(database)

	// Initialize feed syncer and start daily sync
	feedSyncer := sync.NewFeedSyncer(feedRepo, cfg.FeedListPath, cfg.FeedListURL)

	// Start background processes
	bgCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start feed sync in background (runs once immediately, then every 24 hours)
	slog.Info("starting feed sync background task")
	go func() {
		if err := feedSyncer.Run(bgCtx); err != nil {
			slog.Info("feed syncer stopped", slog.Any("error", err))
		}
	}()

	// Initialize HTTP client for communicating with recommender service
	recommenderClient := client.NewRecommenderClient(cfg.RecommenderURL, cfg.InternalAPIKey)

	// Initialize fetcher with configurable interval
	feedFetcher := fetcher.NewFetcher(feedRepo, recommenderClient, cfg.FetchInterval)

	// Start background fetcher (fetches 1 feed per minute)
	slog.Info("starting feed fetcher background task")
	go func() {
		if err := feedFetcher.Run(bgCtx); err != nil {
			slog.Info("fetcher stopped", slog.Any("error", err))
		}
	}()

	// Setup HTTP server for health checks and manual triggers
	internalAuthMiddleware := auth.NewInternalAuthMiddleware(cfg.InternalAPIKey)
	r := api.NewRouter(bgCtx, database, feedRepo, feedFetcher, feedSyncer, internalAuthMiddleware, logger)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down service")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	slog.Info("service ready to accept requests",
		slog.String("port", cfg.Server.Port),
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}
}
