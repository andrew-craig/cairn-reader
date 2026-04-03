package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/logging"
	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/api/handlers"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/service"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the HTTP router for RSS Fetcher Service
func NewRouter(db *database.DB) http.Handler {
	r := chi.NewRouter()

	// Apply global middleware
	r.Use(sharedmw.Recovery)
	r.Use(logging.ChiRequestLogger(slog.Default()))
	r.Use(sharedmw.SecureHeadersRelaxed)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Initialize repositories
	feedRepo := repository.NewFeedRepository(db.DB)
	subscriptionRepo := repository.NewFeedSubscriptionRepository(db.DB)

	// Initialize services
	feedService := service.NewFeedService(feedRepo, subscriptionRepo)

	// Initialize handlers
	subscriptionHandler := handlers.NewSubscriptionHandler(feedService)

	// Health check endpoints (Kubernetes-compatible)
	// Liveness probe - indicates if the process is running
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	// Readiness probe - indicates if the service is ready to accept traffic
	// Returns 503 Service Unavailable if dependencies are unreachable
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection with timeout
		ctx := r.Context()
		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"unhealthy","checks":{"database":"error"}}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"healthy","checks":{"database":"ok"}}`)
	})

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"service":"rss-fetcher-service","version":"0.1.0","status":"running"}`)
	})

	// API v1 routes with /source/rss prefix
	// This service is internal-facing, accessed through Content Service gateway
	r.Route("/api/v1/source/rss", func(r chi.Router) {
		r.Use(sharedmw.RequireHTTPS)
		// User feed subscription routes
		r.Route("/user/{user_id}/subscription", func(r chi.Router) {
			r.Post("/", subscriptionHandler.Subscribe)
			r.Get("/", subscriptionHandler.ListSubscriptions)
			r.Delete("/{feed_id}", subscriptionHandler.Unsubscribe)
		})

		// Feed management routes (generic PATCH for updates)
		r.Route("/feed/{feed_id}", func(r chi.Router) {
			r.Patch("/", subscriptionHandler.UpdateFeed)
		})
	})

	return r
}
