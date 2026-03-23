// Package main is the entry point for the content service.
// The content service is responsible for:
// - Storing and managing article content from various sources
// - Extracting readable content from HTML using readability
// - Sanitizing content for safe display
// - Managing user-specific content metadata
// - Providing bulk content operations
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

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/config"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/database"
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
		ServiceName: "content-service",
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

	// Prepare database config
	port, _ := strconv.Atoi(cfg.Database.Port)
	dbConfig := database.Config{
		Host:     cfg.Database.Host,
		Port:     port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	if err := database.RunMigrations(&dbConfig, migrationsPath); err != nil {
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
	defer db.Close()

	slog.Info("component initialized", slog.String("component", "database"))

	// Initialize Vault client and fetch JWT public key
	slog.Info("component initializing", slog.String("component", "vault"))
	vaultCfg := &auth.VaultConfig{
		Address:   cfg.Vault.Address,
		Token:     cfg.Vault.Token,
		RoleID:    cfg.Vault.RoleID,
		SecretID:  cfg.Vault.SecretID,
		AuthPath:  cfg.Vault.AuthPath,
		Namespace: "",
	}

	vaultClient, err := auth.NewVaultClient(vaultCfg)
	if err != nil {
		slog.Error("failed to create vault client", slog.Any("error", err))
		os.Exit(1)
	}

	// Verify Vault connectivity
	if err := vaultClient.Health(); err != nil {
		slog.Error("vault health check failed", slog.Any("error", err))
		os.Exit(1)
	}

	// Fetch JWT public key from Vault
	publicKey, err := vaultClient.GetPublicKey(cfg.Vault.PublicKeyPath)
	if err != nil {
		slog.Error("failed to fetch public key from vault", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("component initialized", slog.String("component", "vault"))

	// Create JWT validator and middleware
	jwtValidator := auth.NewValidator(publicKey)
	authMiddleware := auth.NewMiddleware(jwtValidator)

	// Create internal service-to-service auth middleware
	internalAuthMiddleware := auth.NewInternalAuthMiddleware(cfg.InternalAPIKey)

	// Create router
	router := api.NewRouter(db, cfg.IngestRSSServiceURL, authMiddleware, internalAuthMiddleware)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("server listening",
			slog.String("addr", server.Addr),
		)
		slog.Info("service ready to accept requests")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	// Give outstanding requests time to complete
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", slog.Any("error", err))
	}

	slog.Info("server exited gracefully")
}
