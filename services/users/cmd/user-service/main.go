package main

import (
	"context"
	"fmt"
	"log"
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
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file in development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Cairn User Service in %s mode on port %s", cfg.Server.Environment, cfg.Server.Port)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Vault client
	log.Println("Connecting to Vault...")
	vaultClient, err := initializeVault(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Vault: %v", err)
	}
	log.Println("✓ Vault connection established")

	// Load JWT keys from Vault
	log.Println("Loading JWT keys from Vault...")
	privateKey, publicKey, err := vaultClient.GetJWTKeys(
		cfg.JWT.PrivateKeyPath,
		cfg.JWT.PublicKeyPath,
	)
	if err != nil {
		log.Fatalf("Failed to load JWT keys from Vault: %v", err)
	}
	log.Println("✓ JWT keys loaded")

	// Initialize database connection
	log.Println("Connecting to database...")
	db, err := initializeDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Database connection established")

	// Run database migrations
	log.Println("Running database migrations...")
	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
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
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Initialize repositories
	log.Println("Initializing repositories...")
	userRepo := database.NewUserRepository(db)
	refreshTokenRepo := database.NewRefreshTokenRepository(db)
	log.Println("✓ Repositories initialized")

	// Initialize auth components
	log.Println("Initializing authentication components...")
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
	log.Println("✓ Authentication components initialized")

	// Initialize services
	log.Println("Initializing services...")
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
	log.Println("✓ Services initialized")

	// Set up key rotation manager
	log.Println("Setting up key rotation manager...")
	keyRotationManager, err := auth.NewKeyRotationManager(vaultClient, auth.KeyRotationConfig{
		PrivateKeyPath:       cfg.JWT.PrivateKeyPath,
		PublicKeyPath:        cfg.JWT.PublicKeyPath,
		RotationInterval:     cfg.JWT.KeyRotationInterval,
		TokenRenewalInterval: cfg.Vault.TokenRenewalInterval,
		OnRotation: func(keyPair *auth.JWTKeyPair) error {
			// Update JWT manager with new keys
			jwtManager.UpdateKeys(keyPair.PrivateKey, keyPair.PublicKey)
			log.Println("✓ JWT keys rotated successfully")
			return nil
		},
	})
	if err != nil {
		log.Fatalf("Failed to create key rotation manager: %v", err)
	}

	// Start key rotation in background
	keyRotationManager.Start(ctx)
	log.Println("✓ Key rotation manager started")

	// Start expired token cleanup scheduler
	log.Println("Starting token cleanup scheduler...")
	go tokenCleanupScheduler(ctx, refreshTokenService)
	log.Println("✓ Token cleanup scheduler started")

	// Set Gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Set up HTTP router
	log.Println("Setting up HTTP router...")
	router := handlers.Router(handlers.RouterConfig{
		DB:                  db,
		VaultClient:         vaultClient,
		AuthService:         authService,
		UserService:         userService,
		JWTManager:          jwtManager,
		AuthRateLimit:       cfg.Security.RateLimitRequests,
		AuthRateLimitWindow: cfg.Security.RateLimitWindow,
	})
	log.Println("✓ HTTP router configured")

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("✓ Server listening on port %s", cfg.Server.Port)
		log.Println("Cairn User Service is ready to accept requests")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel context to stop background tasks
	cancel()

	// Stop key rotation manager
	keyRotationManager.Stop()
	log.Println("✓ Background tasks stopped")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✓ Server exited gracefully")
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
			log.Println("Token cleanup scheduler stopped")
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
		log.Printf("Token cleanup failed: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Cleaned up %d expired refresh tokens", count)
	}
}
