// Package selfhost provides initialization functions for the consolidated self-hosted binary.
// These functions wrap internal packages to make them accessible outside the module.
package selfhost

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"time"

	emailAPI "github.com/cairn-app/cairn-reader/services/read/email/internal/api"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/handlers"
	emailMW "github.com/cairn-app/cairn-reader/services/read/email/internal/api/middleware"
	emailClient "github.com/cairn-app/cairn-reader/services/read/email/internal/client"
	emailConfig "github.com/cairn-app/cairn-reader/services/read/email/internal/config"
	emailDB "github.com/cairn-app/cairn-reader/services/read/email/internal/database"
	emailJobs "github.com/cairn-app/cairn-reader/services/read/email/internal/jobs"
	emailProcessor "github.com/cairn-app/cairn-reader/services/read/email/internal/processor"
	emailRepo "github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
	emailService "github.com/cairn-app/cairn-reader/services/read/email/internal/service"
	emailWorker "github.com/cairn-app/cairn-reader/services/read/email/internal/worker"

	"github.com/go-chi/chi/v5"
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

// EmailConfig holds configuration for the email ingest service.
type EmailConfig struct {
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	EmailDomain    string
	ContentBaseURL string
}

// RunEmailMigrations runs embedded migrations for the email database.
func RunEmailMigrations(cfg EmailConfig, migrations fs.FS) error {
	dbConfig := &emailDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
	return emailDB.RunMigrationsFS(dbConfig, migrations)
}

// MountEmail initializes the email ingest service and mounts routes.
// Returns the underlying *sql.DB for health checks, a cleanup function, and any error.
func MountEmail(ctx context.Context, cfg EmailConfig, r chi.Router, publicKey *rsa.PublicKey, logger *slog.Logger) (*sql.DB, func(), error) {
	dbConfig := &emailDB.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		DBName:          cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    envInt("DB_MAX_CONNS", 3),
		MaxIdleConns:    envInt("DB_MIN_CONNS", 1),
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	db, err := emailDB.NewConnection(dbConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("email db: %w", err)
	}

	// Initialize repositories
	addressRepo := emailRepo.NewAddressRepository(db.DB)
	rawEmailRepo := emailRepo.NewRawEmailRepository(db.DB)
	senderRepo := emailRepo.NewSenderRepository(db.DB)
	apiKeyRepo := emailRepo.NewAPIKeyRepository(db.DB)
	outboxRepo := emailRepo.NewOutboxRepository(db.DB)

	// Initialize services
	addressService := emailService.NewAddressService(addressRepo)
	senderService := emailService.NewSenderService(senderRepo)
	emailSvc := emailService.NewEmailService(addressService, rawEmailRepo)

	// Initialize middleware
	apiKeyAuth := emailMW.NewAPIKeyAuth(apiKeyRepo)
	jwtAuth := emailMW.NewJWTAuth(publicKey)

	// Initialize handlers
	emailCfg := emailConfig.EmailConfig{Domain: cfg.EmailDomain}
	ingestHandler := handlers.NewIngestHandler(emailSvc)
	addressHandler := handlers.NewAddressHandler(addressService, emailCfg)
	senderHandler := handlers.NewSenderHandler(senderService)

	// Mount email API routes
	emailRouter := emailAPI.NewRouter(emailAPI.RouterDeps{
		DB:             db,
		IngestHandler:  ingestHandler,
		AddressHandler: addressHandler,
		SenderHandler:  senderHandler,
		APIKeyAuth:     apiKeyAuth,
		JWTAuth:        jwtAuth,
	})
	r.Handle("/api/v1/source/email", emailRouter)
	r.Handle("/api/v1/source/email/*", emailRouter)

	// Initialize email worker components
	emailCleaner := emailProcessor.NewEmailCleaner()
	contentExtractor := emailProcessor.NewContentExtractor()

	contentClient := emailClient.NewContentServiceClient(emailClient.ContentServiceConfig{
		BaseURL: cfg.ContentBaseURL,
		Timeout: 30 * time.Second,
	})

	emailProcessorWorker := emailWorker.NewEmailProcessorWorker(
		senderService, emailCleaner, contentExtractor,
		rawEmailRepo, outboxRepo,
		emailWorker.EmailProcessorConfig{
			BatchSize:    20,
			WorkerCount:  3,
			PollInterval: 5 * time.Second,
		},
	)

	outboxWorker := emailWorker.NewOutboxWorker(
		outboxRepo, contentClient,
		emailWorker.OutboxWorkerConfig{
			BatchSize:    10,
			PollInterval: 10 * time.Second,
			MaxRetries:   6,
		},
	)

	rawEmailCleanupJob, err := emailJobs.NewRawEmailCleanupJob(rawEmailRepo, "0 5 * * *", 7)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("failed to close db", "error", closeErr)
		}
		return nil, nil, fmt.Errorf("email raw cleanup schedule: %w", err)
	}

	outboxCleanupJob, err := emailJobs.NewOutboxCleanupJob(outboxRepo, "0 6 * * *", 7)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("failed to close db", "error", closeErr)
		}
		return nil, nil, fmt.Errorf("email outbox cleanup schedule: %w", err)
	}

	// Start all workers
	go emailProcessorWorker.Start(ctx)
	go outboxWorker.Start(ctx)
	go rawEmailCleanupJob.Start(ctx)
	go outboxCleanupJob.Start(ctx)

	return db.DB, func() {
		if closeErr := db.Close(); closeErr != nil {
			logger.Error("failed to close db", "error", closeErr)
		}
	}, nil
}
