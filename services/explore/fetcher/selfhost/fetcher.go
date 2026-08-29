// Package selfhost provides initialization functions for the consolidated self-hosted binary.
package selfhost

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	fetcherAPI "github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/api"
	fetcherClient "github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/client"
	fetcherDB "github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/sync"
	"github.com/go-chi/chi/v5"
)

// FetcherConfig holds configuration for the fetcher service.
type FetcherConfig struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	FeedListPath   string
	FeedListURL    string
	FetchInterval  time.Duration
	RecommenderURL string
	InternalAPIKey string
}

// RunMigrations runs embedded migrations for the fetcher database.
func RunMigrations(connString string, migrations fs.FS) error {
	return fetcherDB.RunMigrationsFS(connString, migrations)
}

// Mount initializes the fetcher service and mounts routes.
func Mount(ctx context.Context, cfg FetcherConfig, r chi.Router, internalAuthMiddleware *auth.InternalAuthMiddleware, logger *slog.Logger) (func(), error) {
	dbConfig := &fetcherDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
	}

	database, err := dbConfig.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fetcher db: %w", err)
	}

	feedRepo := fetcherDB.NewFeedRepository(database)
	feedSyncer := sync.NewFeedSyncer(feedRepo, cfg.FeedListPath, cfg.FeedListURL)

	go func() {
		if err := feedSyncer.Run(ctx); err != nil {
			slog.Info("feed syncer stopped", slog.Any("error", err))
		}
	}()

	recommenderClient := fetcherClient.NewRecommenderClient(cfg.RecommenderURL, cfg.InternalAPIKey)
	feedFetcher := fetcher.NewFetcher(feedRepo, recommenderClient, cfg.FetchInterval)

	go func() {
		if err := feedFetcher.Run(ctx); err != nil {
			slog.Info("feed fetcher stopped", slog.Any("error", err))
		}
	}()

	// Mount the same router the standalone binary uses (rather than
	// re-declaring the routes here) so the router-inventory auth ratchet in
	// internal/api/router_auth_test.go covers this deployment path too.
	handler := fetcherAPI.NewRouter(ctx, database, feedRepo, feedFetcher, feedSyncer, internalAuthMiddleware, logger)
	r.Handle("/api/v1/explore/feed", handler)
	r.Handle("/api/v1/explore/feed/*", handler)

	return func() { database.Close() }, nil
}
