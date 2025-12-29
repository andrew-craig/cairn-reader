package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/api/handlers"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/api/middleware"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/database"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/service"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the HTTP router for RSS Fetcher Service
func NewRouter(db *database.DB) http.Handler {
	r := chi.NewRouter()

	// Apply global middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Initialize repositories
	feedRepo := repository.NewFeedRepository(db.DB)
	subscriptionRepo := repository.NewFeedSubscriptionRepository(db.DB)

	// Initialize services
	feedService := service.NewFeedService(feedRepo, subscriptionRepo)

	// Initialize handlers
	subscriptionHandler := handlers.NewSubscriptionHandler(feedService)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		ctx := r.Context()
		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"error","service":"rss-fetcher-service","message":"Database connection failed","error":"%s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"rss-fetcher-service"}`)
	})

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service":"rss-fetcher-service","version":"0.1.0","status":"running"}`)
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// User feed subscription routes
		r.Route("/users/{user_id}/feeds", func(r chi.Router) {
			r.Post("/subscribe", subscriptionHandler.Subscribe)
			r.Get("/", subscriptionHandler.ListSubscriptions)
			r.Delete("/{feed_id}", subscriptionHandler.Unsubscribe)
		})

		// Feed management routes
		r.Route("/feeds/{feed_id}", func(r chi.Router) {
			r.Patch("/enable", subscriptionHandler.EnableFeed)
		})
	})

	return r
}
