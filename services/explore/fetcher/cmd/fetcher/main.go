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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrew-craig/cairn-explore/fetcher/internal/client"
	"github.com/andrew-craig/cairn-explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn-explore/fetcher/internal/fetcher"
	"github.com/andrew-craig/cairn-explore/fetcher/internal/sync"
)

func main() {
	// Load configuration from environment variables with sensible defaults
	port := getEnv("PORT", "8080")
	recommenderURL := getEnv("RECOMMENDER_URL", "http://localhost:8081")
	fetchInterval := getEnvDuration("FETCH_INTERVAL", 60) // Default: 60 seconds (1 feed per minute)
	kagiFeedURL := getEnv("KAGI_FEED_URL", "https://raw.githubusercontent.com/kagisearch/smallweb/main/smallweb.txt")

	// Initialize database connection
	dbConfig := db.NewConfigFromEnv()
	database, err := dbConfig.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize repositories
	feedRepo := db.NewFeedRepository(database)

	// Initialize feed syncer and start daily sync
	feedSyncer := sync.NewFeedSyncer(feedRepo, kagiFeedURL)

	// Start background processes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start feed sync in background (runs once immediately, then every 24 hours)
	go func() {
		if err := feedSyncer.Run(ctx); err != nil {
			log.Printf("Feed syncer stopped: %v", err)
		}
	}()

	// Initialize HTTP client for communicating with recommender service
	recommenderClient := client.NewRecommenderClient(recommenderURL)

	// Initialize fetcher with configurable interval
	feedFetcher := fetcher.NewFetcher(feedRepo, recommenderClient, fetchInterval)

	// Start background fetcher (fetches 1 feed per minute)
	go func() {
		if err := feedFetcher.Run(ctx); err != nil {
			log.Printf("Fetcher stopped: %v", err)
		}
	}()

	// Setup HTTP server for health checks and manual triggers
	// Routes:
	//   GET  /health      - Returns service health status
	//   POST /fetch       - Triggers a single feed fetch (async)
	//   GET  /feeds/stats - Returns feed statistics (total, enabled, disabled, never_fetched)
	//   POST /feeds/sync  - Triggers feed list sync from Kagi (async)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go func() {
			if err := feedFetcher.FetchSingleFeed(ctx); err != nil {
				log.Printf("error in fetch goroutine: %v", err)
			}
		}()
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"fetch triggered"}`))
	})
	mux.HandleFunc("/feeds/stats", func(w http.ResponseWriter, r *http.Request) {
		total, enabled, disabled, neverFetched, err := feedRepo.GetFeedStats(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"total":%d,"enabled":%d,"disabled":%d,"never_fetched":%d}`,
			total, enabled, disabled, neverFetched)))
	})
	mux.HandleFunc("/feeds/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go func() {
			if err := feedSyncer.SyncOnce(ctx); err != nil {
				log.Printf("error in sync goroutine: %v", err)
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

		log.Println("Shutting down fetcher service...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	log.Printf("Fetcher service starting on port %s", port)
	log.Printf("Fetch interval: %v", fetchInterval)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// healthHandler returns a simple health check response for load balancers and monitoring
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"fetcher"}`))
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
