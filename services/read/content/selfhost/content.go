// Package selfhost provides initialization functions for the consolidated self-hosted binary.
package selfhost

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	contentAPI "github.com/cairn-app/cairn-reader/services/read/content/internal/api"
	contentDB "github.com/cairn-app/cairn-reader/services/read/content/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/jobs"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/robfig/cron/v3"
)

// ContentConfig holds configuration for the content service.
type ContentConfig struct {
	DBHost              string
	DBPort              int
	DBUser              string
	DBPassword          string
	DBName              string
	DBSSLMode           string
	IngestRSSServiceURL string
}

// RunMigrations runs embedded migrations for the content database.
func RunMigrations(cfg ContentConfig, migrations fs.FS) error {
	dbConfig := &contentDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
	return contentDB.RunMigrationsFS(dbConfig, migrations)
}

// Mount initializes the content service and mounts routes.
// Returns the underlying *sql.DB for health checks, a cleanup function, and any error.
func Mount(cfg ContentConfig, r chi.Router, authMiddleware *auth.Middleware, internalAuthMiddleware *auth.InternalAuthMiddleware, logger *slog.Logger) (*sql.DB, func(), error) {
	dbConfig := contentDB.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		DBName:          cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
	}

	db, err := contentDB.NewConnection(dbConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("content db: %w", err)
	}

	contentRouter := contentAPI.NewRouter(db, cfg.IngestRSSServiceURL, authMiddleware, internalAuthMiddleware)
	r.Mount("/", contentRouter)

	contentRepo := repository.NewContentRepository(db.DB)
	cleanupJob := jobs.NewCleanupJob(contentRepo, logger)

	scheduler := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(log.New(os.Stdout, "cron: ", log.LstdFlags))))
	_, err = scheduler.AddFunc("0 2 * * *", cleanupJob.Run)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("content cleanup schedule: %w", err)
	}
	scheduler.Start()

	return db.DB, func() {
		cronCtx := scheduler.Stop()
		<-cronCtx.Done()
		db.Close()
	}, nil
}
