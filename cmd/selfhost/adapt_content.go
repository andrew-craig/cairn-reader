package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	contentMigrations "github.com/cairn-app/cairn-reader/services/read/content/migrations"
	contentSelfhost "github.com/cairn-app/cairn-reader/services/read/content/selfhost"
)

func runContentMigrations(cfg *Config) error {
	slog.Info("running content migrations")
	return contentSelfhost.RunMigrations(contentSelfhost.ContentConfig{
		DBHost:     cfg.DB.Host,
		DBPort:     cfg.DB.Port,
		DBUser:     cfg.DB.User,
		DBPassword: cfg.DB.Password,
		DBName:     cfg.DBNameContent,
		DBSSLMode:  cfg.DB.SSLMode,
	}, contentMigrations.FS)
}

func mountContentService(ctx context.Context, cfg *Config, r chi.Router, authMiddleware *auth.Middleware, internalAuthMiddleware *auth.InternalAuthMiddleware, health *healthChecker, logger *slog.Logger) (func(), error) {
	selfhostBaseURL := fmt.Sprintf("http://localhost:%s", cfg.Port)

	sqlDB, closer, err := contentSelfhost.Mount(contentSelfhost.ContentConfig{
		DBHost:                cfg.DB.Host,
		DBPort:                cfg.DB.Port,
		DBUser:                cfg.DB.User,
		DBPassword:            cfg.DB.Password,
		DBName:                cfg.DBNameContent,
		DBSSLMode:             cfg.DB.SSLMode,
		IngestRSSServiceURL:   selfhostBaseURL,
		EmailIngestServiceURL: selfhostBaseURL,
		InternalAPIKey:        cfg.InternalAPIKey,
	}, r, authMiddleware, internalAuthMiddleware, logger)
	if err != nil {
		return nil, err
	}

	health.addDB("content", sqlDB)
	return closer, nil
}
