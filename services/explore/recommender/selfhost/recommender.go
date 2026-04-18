// Package selfhost provides initialization functions for the consolidated self-hosted binary.
package selfhost

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	recAPI "github.com/cairn-app/cairn-reader/services/explore/recommender/internal/api"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/cleanup"
	recDB "github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/recommend"
	"github.com/go-chi/chi/v5"
)

// RecommenderConfig holds configuration for the recommender service.
type RecommenderConfig struct {
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	ArticleRetentionDays int
}

// RunMigrations runs embedded migrations for the recommender database.
func RunMigrations(connString string, migrations fs.FS) error {
	return recDB.RunMigrationsFS(connString, migrations)
}

// Mount initializes the recommender service and mounts routes.
func Mount(ctx context.Context, cfg RecommenderConfig, r chi.Router, authMiddleware *auth.Middleware, logger *slog.Logger) (func(), error) {
	dbConfig := recDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
	}

	database, err := dbConfig.Connect(context.Background())
	if err != nil {
		return nil, fmt.Errorf("recommender db: %w", err)
	}

	articleRepo := recDB.NewArticleRepository(database)
	userRepo := recDB.NewUserRepository(database)
	articleRepo.SetUserRepository(userRepo)
	voteRepo := recDB.NewVoteRepository(database, userRepo)

	recommendEngine := recommend.NewEngine(articleRepo, userRepo)

	cleanupJob := cleanup.NewArticleCleanup(articleRepo, cfg.ArticleRetentionDays, 24*time.Hour, logger)
	cleanupJob.Start()

	server := recAPI.NewServer(database, articleRepo, userRepo, voteRepo, recommendEngine, authMiddleware, logger)
	handler := server.Routes()
	r.Handle("/api/v1/explore", handler)
	r.Handle("/api/v1/explore/*", handler)

	return func() {
		cleanupJob.Stop()
		database.Close()
	}, nil
}
