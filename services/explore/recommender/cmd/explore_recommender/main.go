package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/andrew-craig/cairn/pkg/logging"
	"github.com/andrew-craig/cairn/services/explore/recommender/internal/api"
	"github.com/andrew-craig/cairn/services/explore/recommender/internal/cleanup"
	"github.com/andrew-craig/cairn/services/explore/recommender/internal/db"
	"github.com/andrew-craig/cairn/services/explore/recommender/internal/recommend"
)

func main() {
	// Initialize structured logger
	logger := logging.NewLogger(logging.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "text"),
		ServiceName: "recommender",
	})
	logging.SetDefault(logger)

	port := getEnv("PORT", "8081")

	slog.Info("starting service",
		slog.String("port", port),
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
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "cairn"),
		Password: getEnv("DB_PASSWORD", "cairn_password"),
		DBName:   getEnv("DB_NAME", "cairn_db"),
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
	vaultAddr := getEnv("VAULT_ADDR", "http://localhost:8200")
	vaultToken := getEnv("VAULT_TOKEN", "")
	publicKeyPath := getEnv("JWT_PUBLIC_KEY_PATH", "secret/jwt/public-key")

	slog.Info("connecting to vault",
		slog.String("address", vaultAddr),
	)
	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address: vaultAddr,
		Token:   vaultToken,
	})
	if err != nil {
		slog.Error("failed to create vault client", slog.Any("error", err))
		os.Exit(1)
	}

	// Fetch JWT public key from Vault
	slog.Info("fetching JWT public key from vault",
		slog.String("path", publicKeyPath),
	)
	publicKey, err := vaultClient.GetPublicKey(publicKeyPath)
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
	retentionDays := getEnvAsInt("ARTICLE_RETENTION_DAYS", 90)
	cleanupInterval := 24 * time.Hour // Run cleanup once per day
	cleanupJob := cleanup.NewArticleCleanup(articleRepo, retentionDays, cleanupInterval)
	cleanupJob.Start()
	slog.Info("article cleanup job started",
		slog.Int("retention_days", retentionDays),
		slog.Duration("interval", cleanupInterval),
	)

	// Initialize API server with auth middleware
	server := api.NewServer(database, articleRepo, userRepo, voteRepo, recommendEngine, authMiddleware, logger)
	httpServer := &http.Server{
		Addr:         ":" + port,
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

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	slog.Info("service ready to accept requests",
		slog.String("port", port),
	)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		slog.Warn("invalid environment variable value",
			slog.String("key", key),
			slog.String("value", valueStr),
			slog.Int("default", defaultValue),
		)
		return defaultValue
	}
	return value
}
