// Package main is the entry point for the Email Ingest Worker.
// The worker is responsible for:
// - Processing raw emails from the database
// - Email-specific cleaning (tracking pixels, footers)
// - Content extraction using readability
// - Delivering processed content to the outbox
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/client"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/config"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/jobs"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/processor"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/service"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file in development (before logger initialization)
	_ = godotenv.Load() // Ignore error as .env file is optional

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
		ServiceName: "email-ingest-worker",
	})
	logging.SetDefault(logger)

	slog.Info("starting worker")

	// Run database migrations
	slog.Info("running database migrations")
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		slog.Error("failed to get migrations path", slog.Any("error", err))
		os.Exit(1)
	}

	// Convert database config for migrations
	port, _ := strconv.Atoi(cfg.Database.Port)
	dbConfig := &database.Config{
		Host:     cfg.Database.Host,
		Port:     port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	if err := database.RunMigrations(dbConfig, migrationsPath); err != nil {
		slog.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	// Initialize database connection
	slog.Info("component initializing", slog.String("component", "database"))
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()
	slog.Info("component initialized", slog.String("component", "database"))

	// Initialize repositories
	rawEmailRepo := repository.NewRawEmailRepository(db.DB)
	outboxRepo := repository.NewOutboxRepository(db.DB)
	senderRepo := repository.NewSenderRepository(db.DB)

	// Initialize services and processors
	senderService := service.NewSenderService(senderRepo)
	emailCleaner := processor.NewEmailCleaner()
	contentExtractor := processor.NewContentExtractor()

	// Initialize content service client
	contentClient := client.NewContentServiceClient(client.ContentServiceConfig{
		BaseURL: cfg.ContentService.URL,
		Timeout: cfg.ContentService.Timeout,
	})

	// Initialize workers
	emailProcessor := worker.NewEmailProcessorWorker(
		senderService,
		emailCleaner,
		contentExtractor,
		rawEmailRepo,
		outboxRepo,
		worker.EmailProcessorConfig{
			BatchSize:    20,
			WorkerCount:  cfg.Worker.EmailProcessWorkers,
			PollInterval: cfg.Worker.EmailProcessPollInterval,
		},
	)

	outboxWorker := worker.NewOutboxWorker(
		outboxRepo,
		contentClient,
		worker.OutboxWorkerConfig{
			BatchSize:    10,
			PollInterval: cfg.Worker.OutboxPollInterval,
			MaxRetries:   cfg.Worker.OutboxMaxRetries,
		},
	)

	// Initialize cleanup jobs
	rawEmailCleanupJob, err := jobs.NewRawEmailCleanupJob(
		rawEmailRepo,
		cfg.Worker.RawEmailCleanupCron,
		cfg.Worker.RawEmailRetentionDays,
	)
	if err != nil {
		slog.Error("failed to create raw email cleanup job", slog.Any("error", err))
		os.Exit(1)
	}

	outboxCleanupJob, err := jobs.NewOutboxCleanupJob(
		outboxRepo,
		cfg.Worker.OutboxCleanupCron,
		cfg.Worker.RawEmailRetentionDays,
	)
	if err != nil {
		slog.Error("failed to create outbox cleanup job", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("worker ready")
	slog.Info("- email processor worker")
	slog.Info("- outbox delivery worker")
	slog.Info("- raw email cleanup job", slog.String("schedule", cfg.Worker.RawEmailCleanupCron))
	slog.Info("- outbox cleanup job", slog.String("schedule", cfg.Worker.OutboxCleanupCron))

	// Start health check HTTP server
	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8090"
	}
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	healthMux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := db.PingContext(context.Background()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable", "error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	healthServer := &http.Server{Addr: ":" + healthPort, Handler: healthMux}
	go func() {
		slog.Info("health check server starting", slog.String("port", healthPort))
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health check server failed", slog.Any("error", err))
		}
	}()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Start all workers
	go emailProcessor.Start(ctx)
	go outboxWorker.Start(ctx)
	go rawEmailCleanupJob.Start(ctx)
	go outboxCleanupJob.Start(ctx)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("health check server shutdown failed", slog.Any("error", err))
	}
	<-shutdownCtx.Done()

	slog.Info("worker exited gracefully")
}
