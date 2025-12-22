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
	port := getEnv("PORT", "8080")
	recommenderURL := getEnv("RECOMMENDER_URL", "http://localhost:8081")
	fetchInterval := getEnvDuration("FETCH_INTERVAL", 5*time.Minute)
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

	// Initialize fetcher with feedRepo (Phase 3: one-feed-per-minute)
	feedFetcher := fetcher.NewFetcher(feedRepo, recommenderClient)

	// Start background fetcher (fetches 1 feed per minute)
	go func() {
		if err := feedFetcher.Run(ctx); err != nil {
			log.Printf("Fetcher stopped: %v", err)
		}
	}()

	// Setup HTTP server for health checks and manual triggers
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		go feedFetcher.FetchSingleFeed(ctx)
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
		go feedSyncer.SyncOnce(ctx)
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"fetcher"}`))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value + "s"); err == nil {
			return duration
		}
	}
	return defaultValue
}
