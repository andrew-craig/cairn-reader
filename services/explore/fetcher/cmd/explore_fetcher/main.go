// Package main is the entry point for the fetcher service.
// The fetcher service is responsible for:
// - Syncing feed sources from the Kagi Small Web collection (daily)
// - Fetching RSS feeds on a scheduled basis (1 feed per minute)
// - Submitting fetched articles to the recommender service via HTTP
// - Providing HTTP endpoints for health checks and manual triggers
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn/services/explore/fetcher/internal/client"
	"github.com/andrew-craig/cairn/services/explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn/services/explore/fetcher/internal/fetcher"
	"github.com/andrew-craig/cairn/services/explore/fetcher/internal/sync"
	"github.com/andrew-craig/cairn/pkg/logging"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Initialize structured logger
	logger := logging.NewLogger(logging.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Format:      getEnv("LOG_FORMAT", "text"),
		ServiceName: "fetcher",
	})
	logging.SetDefault(logger)

	// Load configuration from environment variables with sensible defaults
	port := getEnv("PORT", "8080")
	recommenderURL := getEnv("RECOMMENDER_URL", "http://localhost:8081")
	fetchInterval := getEnvDuration("FETCH_INTERVAL", 60) // Default: 60 seconds (1 feed per minute)
	kagiFeedURL := getEnv("KAGI_FEED_URL", "https://raw.githubusercontent.com/kagisearch/smallweb/main/smallweb.txt")

	slog.Info("starting service",
		slog.String("port", port),
		slog.Duration("fetch_interval", fetchInterval),
	)

	// Initialize database connection
	ctx := context.Background()
	slog.Info("connecting to database")
	dbConfig := db.NewConfigFromEnv()
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
	feedSyncer := sync.NewFeedSyncer(feedRepo, kagiFeedURL)

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
	recommenderClient := client.NewRecommenderClient(recommenderURL)

	// Initialize fetcher with configurable interval
	feedFetcher := fetcher.NewFetcher(feedRepo, recommenderClient, fetchInterval)

	// Start background fetcher (fetches 1 feed per minute)
	slog.Info("starting feed fetcher background task")
	go func() {
		if err := feedFetcher.Run(bgCtx); err != nil {
			slog.Info("fetcher stopped", slog.Any("error", err))
		}
	}()

	// Setup HTTP server for health checks and manual triggers
	// Routes (v1 API):
	//   GET  /health/live                      - Liveness check (simple)
	//   GET  /health/ready                     - Readiness check (includes DB)
	//   POST /api/v1/explore/feed/fetch        - Triggers a single feed fetch (async)
	//   GET  /api/v1/explore/feed/stats        - Returns feed statistics (total, enabled, disabled, never_fetched)
	//   POST /api/v1/explore/feed/sync         - Triggers feed list sync from Kagi (async)
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", livenessHandler)
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessHandler(w, r, database)
	})
	mux.HandleFunc("/api/v1/explore/feed/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go func() {
			if err := feedFetcher.FetchSingleFeed(bgCtx); err != nil {
				slog.Error("error in fetch goroutine", slog.Any("error", err))
			}
		}()
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"fetch triggered"}`))
	})
	mux.HandleFunc("/api/v1/explore/feed/stats", func(w http.ResponseWriter, r *http.Request) {
		total, enabled, disabled, neverFetched, err := feedRepo.GetFeedStats(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"total":%d,"enabled":%d,"disabled":%d,"never_fetched":%d}`,
			total, enabled, disabled, neverFetched)))
	})
	mux.HandleFunc("/api/v1/explore/feed/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go func() {
			if err := feedSyncer.SyncOnce(bgCtx); err != nil {
				slog.Error("error in sync goroutine", slog.Any("error", err))
			}
		}()
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"sync triggered"}`))
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
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

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during shutdown", slog.Any("error", err))
		}
	}()

	slog.Info("service ready to accept requests",
		slog.String("port", port),
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
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
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

	response := fmt.Sprintf(`{"status":"%s","timestamp":"%s","checks":%s}`,
		status,
		time.Now().Format(time.RFC3339),
		formatChecks(checks))

	w.WriteHeader(statusCode)
	w.Write([]byte(response))
}

// formatChecks converts a map of checks to JSON format
func formatChecks(checks map[string]string) string {
	if len(checks) == 0 {
		return "{}"
	}

	result := "{"
	first := true
	for key, value := range checks {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":"%s"`, key, value)
		first = false
	}
	result += "}"
	return result
}

// getEnv retrieves an environment variable or returns a default value if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration parses an environment variable as seconds and returns it as a Duration.
// If the variable is not set or cannot be parsed, returns the default value.
// NOTE: The value is expected to be in seconds (e.g., "60" for 60 seconds)
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value + "s"); err == nil {
			return duration
		}
	}
	return defaultValue
}
