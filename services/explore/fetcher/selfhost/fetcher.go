// Package selfhost provides initialization functions for the consolidated self-hosted binary.
package selfhost

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	fetcherClient "github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/client"
	fetcherDB "github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/sync"
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

	r.Route("/api/v1/explore/feed", func(r chi.Router) {
		r.Use(internalAuthMiddleware.RequireInternalAPIKey)
		r.Post("/fetch", func(w http.ResponseWriter, req *http.Request) {
			go func() {
				if err := feedFetcher.FetchSingleFeed(ctx); err != nil {
					slog.Error("error in fetch goroutine", slog.Any("error", err))
				}
			}()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"status":"fetch triggered"}}`))
		})

		r.Get("/stats", func(w http.ResponseWriter, req *http.Request) {
			total, enabled, disabled, neverFetched, err := feedRepo.GetFeedStats(req.Context())
			if err != nil {
				http.Error(w, "Failed to retrieve feed statistics", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"data":{"total":%d,"enabled":%d,"disabled":%d,"never_fetched":%d}}`,
				total, enabled, disabled, neverFetched)
		})

		r.Post("/sync", func(w http.ResponseWriter, req *http.Request) {
			go func() {
				if err := feedSyncer.SyncOnce(ctx); err != nil {
					slog.Error("error in sync goroutine", slog.Any("error", err))
				}
			}()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"data":{"status":"sync triggered"}}`))
		})
	})

	return func() { database.Close() }, nil
}
