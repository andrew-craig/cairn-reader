package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	rssMigrations "github.com/andrew-craig/cairn-reader/services/read/fetcher/migrations"
	rssSelfhost "github.com/andrew-craig/cairn-reader/services/read/fetcher/selfhost"
)

func runRSSMigrations(cfg *Config) error {
	slog.Info("running RSS migrations")
	return rssSelfhost.RunMigrations(rssSelfhost.RSSConfig{
		DBHost:     cfg.DB.Host,
		DBPort:     cfg.DB.Port,
		DBUser:     cfg.DB.User,
		DBPassword: cfg.DB.Password,
		DBName:     cfg.DBNameRSS,
		DBSSLMode:  cfg.DB.SSLMode,
	}, rssMigrations.FS)
}

func mountIngestRSS(ctx context.Context, cfg *Config, r chi.Router, internalAuthMiddleware *auth.InternalAuthMiddleware, health *healthChecker, logger *slog.Logger) (func(), error) {
	sqlDB, closer, err := rssSelfhost.Mount(rssSelfhost.RSSConfig{
		DBHost:         cfg.DB.Host,
		DBPort:         cfg.DB.Port,
		DBUser:         cfg.DB.User,
		DBPassword:     cfg.DB.Password,
		DBName:         cfg.DBNameRSS,
		DBSSLMode:      cfg.DB.SSLMode,
		ContentBaseURL: fmt.Sprintf("http://localhost:%s", cfg.Port),
		InternalAPIKey: cfg.InternalAPIKey,
	}, r, internalAuthMiddleware, logger)
	if err != nil {
		return nil, err
	}

	health.addDB("rss", sqlDB)
	return closer, nil
}
