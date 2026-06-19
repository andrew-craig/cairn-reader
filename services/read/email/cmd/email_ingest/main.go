// Package main is the entry point for the Email Ingest service.
// The email ingest service is responsible for:
// - Receiving emails from Cloudflare Email Worker
// - Managing user email addresses
// - Storing raw emails for processing
// - Providing email address and sender management API
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
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/handlers"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/middleware"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/config"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/service"
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
		ServiceName: "email-ingest-service",
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

	// Convert database config for migrations
	port, _ := strconv.Atoi(cfg.Database.Port)
	dbConfig := &database.Config{
		Host:             cfg.Database.Host,
		Port:             port,
		User:             cfg.Database.User,
		Password:         cfg.Database.Password,
		DBName:           cfg.Database.DBName,
		SSLMode:          cfg.Database.SSLMode,
		StatementTimeout: cfg.Database.StatementTimeout,
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

	// Initialize Vault client and fetch JWT public key for authentication
	slog.Info("connecting to vault",
		slog.String("address", cfg.Auth.VaultAddr),
	)
	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address:  cfg.Auth.VaultAddr,
		Token:    cfg.Auth.VaultToken,
		RoleID:   cfg.Auth.VaultRoleID,
		SecretID: cfg.Auth.VaultSecretID,
		AuthPath: cfg.Auth.VaultAuthPath,
	})
	if err != nil {
		slog.Error("failed to create vault client", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("fetching JWT public key from vault",
		slog.String("path", cfg.Auth.JWTPublicKeyPath),
	)
	publicKey, err := vaultClient.GetPublicKey(cfg.Auth.JWTPublicKeyPath)
	if err != nil {
		slog.Error("failed to get JWT public key from vault", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("component initialized", slog.String("component", "vault"))

	// Initialize repositories
	addressRepo := repository.NewAddressRepository(db.DB)
	rawEmailRepo := repository.NewRawEmailRepository(db.DB)
	senderRepo := repository.NewSenderRepository(db.DB)
	apiKeyRepo := repository.NewAPIKeyRepository(db.DB)

	// Initialize services
	addressService := service.NewAddressService(addressRepo)
	senderService := service.NewSenderService(senderRepo)
	emailService := service.NewEmailService(addressService, rawEmailRepo)

	// Initialize middleware
	apiKeyAuth := middleware.NewAPIKeyAuth(apiKeyRepo)
	jwtAuth := middleware.NewJWTAuth(publicKey)

	// Initialize handlers
	ingestHandler := handlers.NewIngestHandler(emailService)
	addressHandler := handlers.NewAddressHandler(addressService, cfg.Email)
	senderHandler := handlers.NewSenderHandler(senderService)

	// Create router
	router := api.NewRouter(api.RouterDeps{
		DB:             db,
		IngestHandler:  ingestHandler,
		AddressHandler: addressHandler,
		SenderHandler:  senderHandler,
		APIKeyAuth:     apiKeyAuth,
		JWTAuth:        jwtAuth,
	})

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
