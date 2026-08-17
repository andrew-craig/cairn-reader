// Package selfhost provides initialization functions for the consolidated self-hosted binary.
// These functions wrap internal packages to make them accessible outside the module.
package selfhost

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	userAuth "github.com/cairn-app/cairn-reader/services/users/internal/auth"
	userDB "github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/handlers"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"

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

// UsersConfig holds configuration for the user service.
type UsersConfig struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey

	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration
	BcryptCost       int
}

// RunUsersMigrations runs embedded migrations for the users database.
func RunUsersMigrations(cfg UsersConfig, migrations fs.FS) error {
	dbCfg := &userDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Database: cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
	return userDB.RunMigrationsFS(dbCfg, migrations)
}

// MountUsers initializes the user service and mounts routes on the provided router.
// Returns a cleanup function and any error.
func MountUsers(ctx context.Context, cfg UsersConfig, r chi.Router, logger *slog.Logger) (func(), error) {
	dbCfg := &userDB.Config{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		Database:        cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		MaxOpenConns:    int32(envInt("DB_MAX_CONNS", 3)),
		MaxIdleConns:    int32(envInt("DB_MIN_CONNS", 1)),
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := userDB.New(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("users db: %w", err)
	}

	// Initialize repositories
	userRepo := userDB.NewUserRepository(db)
	refreshTokenRepo := userDB.NewRefreshTokenRepository(db)

	// Initialize auth components
	passwordHasher := userAuth.NewPasswordHasher(cfg.BcryptCost)
	jwtManager := userAuth.NewJWTManagerWithConfig(userAuth.JWTManagerConfig{
		PrivateKey: cfg.PrivateKey,
		PublicKey:  cfg.PublicKey,
		Expiry:     cfg.JWTAccessExpiry,
		Issuer:     "cairn-user-service",
		Audience:   "cairn-api",
	})

	refreshTokenService := userAuth.NewRefreshTokenService(
		refreshTokenRepo,
		cfg.JWTRefreshExpiry,
	)

	// Initialize services
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            userRepo,
		RefreshTokenService: refreshTokenService,
		JWTManager:          jwtManager,
		PasswordHasher:      passwordHasher,
		PasswordMinLength:   8,
		RequireComplexity:   false,
	})

	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          userRepo,
		RefreshTokenRepo:  refreshTokenRepo,
		PasswordHasher:    passwordHasher,
		PasswordMinLength: 8,
		RequireComplexity: false,
	})

	emailVerificationService := services.NewEmailVerificationService(userRepo)

	// Start token cleanup in background
	go tokenCleanupLoop(ctx, refreshTokenService)

	// Mount routes on master router
	router := handlers.Router(handlers.RouterConfig{
		DB:                       db,
		VaultClient:              nil, // No vault in selfhost mode
		AuthService:              authService,
		UserService:              userService,
		EmailVerificationService: emailVerificationService,
		JWTManager:               jwtManager,
		AuthRateLimit:            10,
		AuthRateLimitWindow:      1 * time.Minute,
		Logger:                   logger,
	})

	// Register at specific path prefixes rather than mounting at "/" to avoid conflicts
	// when multiple services share the same chi router in the selfhost binary.
	r.Handle("/api/v1/auth", router)
	r.Handle("/api/v1/auth/*", router)
	r.Handle("/api/v1/user", router)
	r.Handle("/api/v1/user/*", router)

	return func() { db.Close() }, nil
}

func tokenCleanupLoop(ctx context.Context, svc *userAuth.RefreshTokenService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run initial cleanup
	if count, err := svc.CleanupExpiredTokens(ctx); err == nil && count > 0 {
		slog.Info("cleaned up expired tokens", slog.Int64("count", count))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if count, err := svc.CleanupExpiredTokens(ctx); err == nil && count > 0 {
				slog.Info("cleaned up expired tokens", slog.Int64("count", count))
			}
		}
	}
}

// StubHandler returns an http.Handler that does nothing (used to suppress per-service health routes).
var _ http.Handler = http.DefaultServeMux
