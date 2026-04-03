// Package main is the entry point for the fetcher service.
// The fetcher service is responsible for:
// - Syncing feed sources from the Kagi Small Web collection (daily)
// - Fetching RSS feeds on a scheduled basis (1 feed per minute)
// - Submitting fetched articles to the recommender service via HTTP
// - Providing HTTP endpoints for health checks and manual triggers
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/config"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/sync"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
		ServiceName: "fetcher",
	})
	logging.SetDefault(logger)

	slog.Info("starting service",
		slog.String("port", cfg.Server.Port),
		slog.Duration("fetch_interval", cfg.FetchInterval),
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

	// Initialize database connection
	ctx := context.Background()
	slog.Info("connecting to database")
	dbConfig := &db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}
	database, err := dbConfig.Connect(ctx)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer database.Close()
	slog.Info("component initialized", slog.String("component", "database"))

	// Initialize repositories
	feedRepo := db.NewFeedRepository(database)

	// Initialize feed syncer and start daily sync
	feedSyncer := sync.NewFeedSyncer(feedRepo, cfg.KagiFeedURL)

	// Start background processes
	bgCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start feed sync in background (runs once immediately, then every 24 hours)
	slog.Info("starting feed sync background task")
	go func() {
		if err := feedSyncer.Run(bgCtx); err != nil {
			slog.Info("feed syncer stopped", slog.Any("error", err))
		}
	}()

	// Initialize HTTP client for communicating with recommender service
	recommenderClient := client.NewRecommenderClient(cfg.RecommenderURL)

	// Initialize fetcher with configurable interval
	feedFetcher := fetcher.NewFetcher(feedRepo, recommenderClient, cfg.FetchInterval)

	// Start background fetcher (fetches 1 feed per minute)
	slog.Info("starting feed fetcher background task")
	go func() {
		if err := feedFetcher.Run(bgCtx); err != nil {
			slog.Info("fetcher stopped", slog.Any("error", err))
		}
	}()

	// Setup HTTP server for health checks and manual triggers
	r := chi.NewRouter()

	// Global middleware
	r.Use(sharedmw.Recovery)
	r.Use(logging.ChiRequestLogger(logger))
	r.Use(sharedmw.SecureHeadersRelaxed)

	// Health check endpoints
	r.Get("/health/live", livenessHandler)
	r.Head("/health/live", livenessHandler)
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessHandler(w, r, database)
	})
	r.Head("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessHandler(w, r, database)
	})

	// API v1 routes
	r.Route("/api/v1/explore/feed", func(r chi.Router) {
		r.Use(sharedmw.RequireHTTPS)
		r.Post("/fetch", func(w http.ResponseWriter, r *http.Request) {
			go func() {
				if err := feedFetcher.FetchSingleFeed(bgCtx); err != nil {
					slog.Error("error in fetch goroutine", slog.Any("error", err))
				}
			}()
			api.WriteSuccess(w, http.StatusAccepted, map[string]string{"status": "fetch triggered"}, "v1")
		})

		r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			total, enabled, disabled, neverFetched, err := feedRepo.GetFeedStats(r.Context())
			if err != nil {
				api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "Failed to retrieve feed statistics", nil, "v1")
				return
			}
			stats := map[string]int{
				"total":         total,
				"enabled":       enabled,
				"disabled":      disabled,
				"never_fetched": neverFetched,
			}
			api.WriteSuccess(w, http.StatusOK, stats, "v1")
		})

		r.Post("/sync", func(w http.ResponseWriter, r *http.Request) {
			go func() {
				if err := feedSyncer.SyncOnce(bgCtx); err != nil {
					slog.Error("error in sync goroutine", slog.Any("error", err))
				}
			}()
			api.WriteSuccess(w, http.StatusAccepted, map[string]string{"status": "sync triggered"}, "v1")
		})
	})

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutting down service")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	slog.Info("service ready to accept requests",
		slog.String("port", cfg.Server.Port),
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}
}

// livenessHandler returns a simple liveness check (process is running)
// Used by orchestrators to determine if the service process should be restarted
func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
	}{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("error encoding response", "error", err)
	}
}

// readinessHandler checks if the service is ready to accept traffic
// Used by load balancers to determine if traffic should be routed to this instance
// Returns 503 Service Unavailable if dependencies are unreachable
func readinessHandler(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity with 5s timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	status := "healthy"
	statusCode := http.StatusOK

	// Ping database
	if err := db.Ping(ctx); err != nil {
		checks["database"] = "error"
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
		slog.Warn("database health check failed", slog.Any("error", err))
	} else {
		checks["database"] = "ok"
	}

	response := struct {
		Status    string            `json:"status"`
		Timestamp string            `json:"timestamp"`
		Checks    map[string]string `json:"checks"`
	}{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("error encoding response", "error", err)
	}
}
