package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/api"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/cleanup"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/config"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/recommend"
)

func main() {
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
		ServiceName: "recommender",
	})
	logging.SetDefault(logger)

	slog.Info("starting service",
		slog.String("port", cfg.Server.Port),
	)

	// Run database migrations
	slog.Info("running database migrations")
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		slog.Error("failed to get migrations path", slog.Any("error", err))
		os.Exit(1)
	}

	if err := db.RunMigrations(migrationsPath); err != nil {
		slog.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	// Database configuration
	dbConfig := db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}

	// Connect to database
	ctx := context.Background()
	slog.Info("connecting to database",
		slog.String("host", dbConfig.Host),
	)
	database, err := dbConfig.Connect(ctx)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()

	// Initialize Vault client and fetch JWT public key for authentication
	slog.Info("connecting to vault",
		slog.String("address", cfg.Vault.Address),
	)
	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address: cfg.Vault.Address,
		Token:   cfg.Vault.Token,
	})
	if err != nil {
		slog.Error("failed to create vault client", slog.Any("error", err))
		os.Exit(1)
	}

	// Fetch JWT public key from Vault
	slog.Info("fetching JWT public key from vault",
		slog.String("path", cfg.Vault.PublicKeyPath),
	)
	publicKey, err := vaultClient.GetPublicKey(cfg.Vault.PublicKeyPath)
	if err != nil {
		slog.Error("failed to get JWT public key from vault", slog.Any("error", err))
		os.Exit(1)
	}

	// Create JWT validator and auth middleware
	validator := auth.NewValidator(publicKey)
	authMiddleware := auth.NewMiddleware(validator)
	slog.Info("JWT authentication middleware initialized")

	// Initialize repositories
	articleRepo := db.NewArticleRepository(database)
	userRepo := db.NewUserRepository(database)
	articleRepo.SetUserRepository(userRepo) // Set user repo for user ID conversion
	voteRepo := db.NewVoteRepository(database, userRepo)

	// Initialize recommendation engine
	recommendEngine := recommend.NewEngine(articleRepo, userRepo)

	// Initialize article cleanup job
	cleanupInterval := 24 * time.Hour // Run cleanup once per day
	cleanupJob := cleanup.NewArticleCleanup(articleRepo, cfg.ArticleRetentionDays, cleanupInterval, logger)
	cleanupJob.Start()
	slog.Info("article cleanup job started",
		slog.Int("retention_days", cfg.ArticleRetentionDays),
		slog.Duration("interval", cleanupInterval),
	)

	// Initialize API server with auth middleware
	server := api.NewServer(database, articleRepo, userRepo, voteRepo, recommendEngine, authMiddleware, logger)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      server.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down service")

		// Stop cleanup job
		cleanupJob.Stop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	slog.Info("service ready to accept requests",
		slog.String("port", cfg.Server.Port),
	)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}
}
