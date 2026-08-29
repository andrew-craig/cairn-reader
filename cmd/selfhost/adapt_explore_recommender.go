package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	recMigrations "github.com/andrew-craig/cairn-reader/services/explore/recommender/migrations"
	recSelfhost "github.com/andrew-craig/cairn-reader/services/explore/recommender/selfhost"
)

func runRecommenderMigrations(cfg *Config) error {
	slog.Info("running recommender migrations")
	connStr := cfg.DB.ConnString(cfg.DBNameRecommender)
	return recSelfhost.RunMigrations(connStr, recMigrations.FS)
}

func mountExploreRecommender(ctx context.Context, cfg *Config, r chi.Router, authMiddleware *auth.Middleware, internalAuthMiddleware *auth.InternalAuthMiddleware, health *healthChecker, logger *slog.Logger) (func(), error) {
	return recSelfhost.Mount(ctx, recSelfhost.RecommenderConfig{
		DBHost:               cfg.DB.Host,
		DBPort:               fmt.Sprintf("%d", cfg.DB.Port),
		DBUser:               cfg.DB.User,
		DBPassword:           cfg.DB.Password,
		DBName:               cfg.DBNameRecommender,
		ArticleRetentionDays: cfg.ArticleRetentionDays,
	}, r, authMiddleware, internalAuthMiddleware, logger)
}
