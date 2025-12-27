package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn-core/user-service/internal/auth"
	"github.com/andrew-craig/cairn-core/user-service/internal/config"
	"github.com/andrew-craig/cairn-core/user-service/internal/database"
	"github.com/andrew-craig/cairn-core/user-service/internal/handlers"
	"github.com/andrew-craig/cairn-core/user-service/internal/services"
	"github.com/andrew-craig/cairn-core/user-service/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file in development (before logger initialization)
	_ = godotenv.Load() // Ignore error as .env file is optional

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize structured logger
	logger := logging.NewLogger(logging.Config{
		Level:       cfg.Logging.Level,
		Format:      cfg.Logging.Format,
		ServiceName: "user-service",
	})
	logging.SetDefault(logger)

	slog.Info("starting service",
		slog.String("environment", cfg.Server.Environment),
		slog.String("port", cfg.Server.Port),
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Vault client
	slog.Info("component initializing", slog.String("component", "vault"))
	vaultClient, err := initializeVault(cfg)
	if err != nil {
		slog.Error("failed to initialize vault", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("component initialized", slog.String("component", "vault"))

	// Load JWT keys from Vault
	slog.Info("loading JWT keys from vault")
	privateKey, publicKey, err := vaultClient.GetJWTKeys(
		cfg.JWT.PrivateKeyPath,
		cfg.JWT.PublicKeyPath,
	)
	if err != nil {
		slog.Error("failed to load JWT keys", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("JWT keys loaded")

	// Initialize database connection
	slog.Info("component initializing",
		slog.String("component", "database"),
		slog.String("host", cfg.Database.Host),
	)
	db, err := initializeDatabase(cfg)
	if err != nil {
		slog.Error("failed to initialize database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("component initialized", slog.String("component", "database"))

	// Run database migrations
	slog.Info("running database migrations")
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		slog.Error("failed to get migrations path", slog.Any("error", err))
		os.Exit(1)
	}

	dbConfig := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    int32(cfg.Database.MaxOpenConns),
		MaxIdleConns:    int32(cfg.Database.MaxIdleConns),
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	if err := database.RunMigrations(dbConfig, migrationsPath); err != nil {
		slog.Error("failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("database migrations completed")

	// Initialize repositories
	slog.Info("initializing repositories")
	userRepo := database.NewUserRepository(db)
	refreshTokenRepo := database.NewRefreshTokenRepository(db)
	slog.Info("repositories initialized")

	// Initialize auth components
	slog.Info("initializing authentication components")
	passwordHasher := auth.NewPasswordHasher(cfg.Security.BcryptCost)

	jwtManager := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     cfg.JWT.AccessTokenExpiry,
		Issuer:     "cairn-user-service",
		Audience:   "cairn-api",
	})

	refreshTokenService := auth.NewRefreshTokenService(
		refreshTokenRepo,
		cfg.JWT.RefreshTokenExpiry,
	)
	slog.Info("authentication components initialized")

	// Initialize services
	slog.Info("initializing services")
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            userRepo,
		RefreshTokenService: refreshTokenService,
		JWTManager:          jwtManager,
		PasswordHasher:      passwordHasher,
		PasswordMinLength:   cfg.Security.MinPasswordLength,
		RequireComplexity:   cfg.Security.RequirePasswordComplexity,
	})

	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          userRepo,
		RefreshTokenRepo:  refreshTokenRepo,
		PasswordHasher:    passwordHasher,
		PasswordMinLength: cfg.Security.MinPasswordLength,
		RequireComplexity: cfg.Security.RequirePasswordComplexity,
	})
	slog.Info("services initialized")

	// Set up key rotation manager
	slog.Info("setting up key rotation manager")
	keyRotationManager, err := auth.NewKeyRotationManager(vaultClient, auth.KeyRotationConfig{
		PrivateKeyPath:       cfg.JWT.PrivateKeyPath,
		PublicKeyPath:        cfg.JWT.PublicKeyPath,
		RotationInterval:     cfg.JWT.KeyRotationInterval,
		TokenRenewalInterval: cfg.Vault.TokenRenewalInterval,
		OnRotation: func(keyPair *auth.JWTKeyPair) error {
			// Update JWT manager with new keys
			jwtManager.UpdateKeys(keyPair.PrivateKey, keyPair.PublicKey)
			slog.Info("JWT keys rotated successfully")
			return nil
		},
	})
	if err != nil {
		slog.Error("failed to create key rotation manager", slog.Any("error", err))
		os.Exit(1)
	}

	// Start key rotation in background
	keyRotationManager.Start(ctx)
	slog.Info("key rotation manager started")

	// Start expired token cleanup scheduler
	slog.Info("starting token cleanup scheduler")
	go tokenCleanupScheduler(ctx, refreshTokenService)
	slog.Info("token cleanup scheduler started")

	// Set Gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Set up HTTP router
	slog.Info("setting up HTTP router")
	router := handlers.Router(handlers.RouterConfig{
		DB:                  db,
		VaultClient:         vaultClient,
		AuthService:         authService,
		UserService:         userService,
		JWTManager:          jwtManager,
		AuthRateLimit:       cfg.Security.RateLimitRequests,
		AuthRateLimitWindow: cfg.Security.RateLimitWindow,
		Logger:              logger,
	})
	slog.Info("HTTP router configured")

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("server listening",
			slog.String("port", cfg.Server.Port),
		)
		slog.Info("service ready to accept requests")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	// Cancel context to stop background tasks
	cancel()

	// Stop key rotation manager
	keyRotationManager.Stop()
	slog.Info("background tasks stopped")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("server exited gracefully")
}

// initializeVault creates and configures the Vault client
func initializeVault(cfg *config.Config) (*auth.VaultClient, error) {
	vaultCfg := &auth.VaultConfig{
		Address:   cfg.Vault.Address,
		Token:     cfg.Vault.Token,
		Namespace: cfg.Vault.Namespace,
		RoleID:    cfg.Vault.RoleID,
		SecretID:  cfg.Vault.SecretID,
		AuthPath:  cfg.Vault.AuthPath,
	}

	vaultClient, err := auth.NewVaultClient(vaultCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Verify Vault health
	if err := vaultClient.Health(); err != nil {
		return nil, fmt.Errorf("vault health check failed: %w", err)
	}

	return vaultClient, nil
}

// initializeDatabase creates and configures the database connection
func initializeDatabase(cfg *config.Config) (*database.DB, error) {
	dbConfig := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    int32(cfg.Database.MaxOpenConns),
		MaxIdleConns:    int32(cfg.Database.MaxIdleConns),
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	db, err := database.New(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	// Verify database health
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Health(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	return db, nil
}

// tokenCleanupScheduler runs periodic cleanup of expired refresh tokens
func tokenCleanupScheduler(ctx context.Context, refreshTokenService *auth.RefreshTokenService) {
	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run initial cleanup
	runTokenCleanup(ctx, refreshTokenService)

	for {
		select {
		case <-ctx.Done():
			slog.Info("token cleanup scheduler stopped")
			return
		case <-ticker.C:
			runTokenCleanup(ctx, refreshTokenService)
		}
	}
}

// runTokenCleanup executes the cleanup operation
func runTokenCleanup(ctx context.Context, refreshTokenService *auth.RefreshTokenService) {
	count, err := refreshTokenService.CleanupExpiredTokens(ctx)
	if err != nil {
		slog.Warn("token cleanup failed", slog.Any("error", err))
		return
	}

	if count > 0 {
		slog.Info("cleaned up expired tokens",
			slog.Int64("count", count),
		)
	}
}
