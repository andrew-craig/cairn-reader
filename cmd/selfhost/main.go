package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := loadConfig()

	logger := logging.NewLogger(logging.Config{
		Level:       cfg.LogLevel,
		Format:      cfg.LogFormat,
		ServiceName: "cairn-selfhost",
	})
	logging.SetDefault(logger)

	slog.Info("starting cairn self-hosted server", slog.String("port", cfg.Port))

	// Load JWT keys (file-based, auto-generate if needed)
	slog.Info("loading JWT keys")
	keyProvider := &auth.FileKeyProvider{
		PrivateKeyPath: cfg.JWTPrivateKeyFile,
		PublicKeyPath:  cfg.JWTPublicKeyFile,
		AutoGenerate:   cfg.JWTAutoGenerate,
	}

	privateKey, publicKey, err := keyProvider.GetKeys()
	if err != nil {
		slog.Error("failed to load JWT keys", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("JWT keys loaded")

	// Run all migrations
	slog.Info("running database migrations")
	if err := runAllMigrations(cfg); err != nil {
		slog.Error("migration failed", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("all migrations completed")

	// Create shared auth middleware (for services that validate JWT)
	jwtValidator := auth.NewValidator(publicKey)
	authMiddleware := auth.NewMiddleware(jwtValidator)
	internalAuthMiddleware := auth.NewInternalAuthMiddleware(cfg.InternalAPIKey)

	// Build master router
	r := chi.NewRouter()
	r.Use(sharedmw.Recovery)
	r.Use(logging.ChiRequestLogger(logger))
	r.Use(sharedmw.SecureHeadersRelaxed)

	// Health checker for aggregated health endpoints
	health := newHealthChecker()

	// Create context for background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Closers to defer
	var closers []func()

	// Initialize and mount each service
	slog.Info("initializing user service")
	userCloser, err := mountUserService(ctx, cfg, r, privateKey, publicKey, health, logger)
	if err != nil {
		slog.Error("failed to initialize user service", slog.Any("error", err))
		os.Exit(1)
	}
	closers = append(closers, userCloser)

	slog.Info("initializing explore recommender")
	recCloser, err := mountExploreRecommender(ctx, cfg, r, authMiddleware, health, logger)
	if err != nil {
		slog.Error("failed to initialize explore recommender", slog.Any("error", err))
		os.Exit(1)
	}
	closers = append(closers, recCloser)

	slog.Info("initializing explore fetcher")
	fetchCloser, err := mountExploreFetcher(ctx, cfg, r, health, logger)
	if err != nil {
		slog.Error("failed to initialize explore fetcher", slog.Any("error", err))
		os.Exit(1)
	}
	closers = append(closers, fetchCloser)

	slog.Info("initializing content service")
	contentCloser, err := mountContentService(ctx, cfg, r, authMiddleware, internalAuthMiddleware, health, logger)
	if err != nil {
		slog.Error("failed to initialize content service", slog.Any("error", err))
		os.Exit(1)
	}
	closers = append(closers, contentCloser)

	slog.Info("initializing ingest RSS service")
	rssCloser, err := mountIngestRSS(ctx, cfg, r, health, logger)
	if err != nil {
		slog.Error("failed to initialize ingest RSS service", slog.Any("error", err))
		os.Exit(1)
	}
	closers = append(closers, rssCloser)

	slog.Info("initializing email ingest service")
	emailCloser, err := mountEmailIngest(ctx, cfg, r, publicKey, health, logger)
	if err != nil {
		slog.Error("failed to initialize email ingest service", slog.Any("error", err))
		os.Exit(1)
	}
	closers = append(closers, emailCloser)

	// Mount aggregated health endpoints
	r.Get("/health/live", health.livenessHandler)
	r.Head("/health/live", health.livenessHandler)
	r.Get("/health/ready", health.readinessHandler)
	r.Head("/health/ready", health.readinessHandler)

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server listening", slog.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", slog.Any("error", err))
	}

	// Close all database connections
	for _, closer := range closers {
		closer()
	}

	slog.Info("shutdown complete")
}

// runAllMigrations runs embedded migrations for all 6 databases.
func runAllMigrations(cfg *Config) error {
	if err := runUsersMigrations(cfg); err != nil {
		return err
	}
	if err := runRecommenderMigrations(cfg); err != nil {
		return err
	}
	if err := runFetcherMigrations(cfg); err != nil {
		return err
	}
	if err := runContentMigrations(cfg); err != nil {
		return err
	}
	if err := runRSSMigrations(cfg); err != nil {
		return err
	}
	if err := runEmailMigrations(cfg); err != nil {
		return err
	}
	return nil
}

