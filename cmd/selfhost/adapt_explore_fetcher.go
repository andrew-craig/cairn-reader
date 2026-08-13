package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	fetcherMigrations "github.com/cairn-app/cairn-reader/services/explore/fetcher/migrations"
	fetcherSelfhost "github.com/cairn-app/cairn-reader/services/explore/fetcher/selfhost"
)

func runFetcherMigrations(cfg *Config) error {
	slog.Info("running fetcher migrations")
	connStr := cfg.DB.ConnString(cfg.DBNameFetcher)
	return fetcherSelfhost.RunMigrations(connStr, fetcherMigrations.FS)
}

func mountExploreFetcher(ctx context.Context, cfg *Config, r chi.Router, internalAuthMiddleware *auth.InternalAuthMiddleware, health *healthChecker, logger *slog.Logger) (func(), error) {
	return fetcherSelfhost.Mount(ctx, fetcherSelfhost.FetcherConfig{
		DBHost:         cfg.DB.Host,
		DBPort:         fmt.Sprintf("%d", cfg.DB.Port),
		DBUser:         cfg.DB.User,
		DBPassword:     cfg.DB.Password,
		DBName:         cfg.DBNameFetcher,
		FeedListPath:   cfg.FeedListPath,
		FeedListURL:    cfg.FeedListURL,
		FetchInterval:  cfg.FetchInterval,
		RecommenderURL: fmt.Sprintf("http://localhost:%s", cfg.Port),
		InternalAPIKey: cfg.InternalAPIKey,
	}, r, internalAuthMiddleware, logger)
}
