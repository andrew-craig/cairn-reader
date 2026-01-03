package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/andrew-craig/cairn/pkg/logging"
	"github.com/andrew-craig/cairn/services/read/content/internal/api/handlers"
	"github.com/andrew-craig/cairn/services/read/content/internal/api/middleware"
	"github.com/andrew-craig/cairn/services/read/content/internal/database"
	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates and configures the HTTP router
func NewRouter(db *database.DB) http.Handler {
	r := chi.NewRouter()

	// Apply global middleware
	r.Use(middleware.Recovery)
	r.Use(logging.ChiRequestLogger(slog.Default()))
	r.Use(middleware.ValidateJSON)

	// Initialize repositories
	contentRepo := repository.NewContentRepository(db.DB)
	userContentRepo := repository.NewUserContentRepository(db.DB)

	// Initialize services
	contentService := service.NewContentService(contentRepo, db.DB)
	urlDetector := service.NewURLDetector()

	// Initialize handlers
	contentHandler := handlers.NewContentHandler(contentService)
	userContentHandler := handlers.NewUserContentHandler(userContentRepo, contentRepo)
	bulkHandler := handlers.NewBulkHandler(contentService, userContentRepo, contentRepo)
	detectionHandler := handlers.NewDetectionHandler(urlDetector)

	// Health check endpoints (Kubernetes-compatible)
	// Liveness probe - indicates if the process is running
	r.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	// Readiness probe - indicates if the service is ready to accept traffic
	// Returns 503 Service Unavailable if dependencies are unreachable
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection with timeout
		ctx := r.Context()
		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"unhealthy","checks":{"database":"error"}}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","checks":{"database":"ok"}}`)
	})

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service":"content-service","version":"0.4.0","status":"Phase 1.4 - Bulk Operations complete"}`)
	})

	// API v1 routes - all under /api/v1/content prefix for consistent service boundary
	r.Route("/api/v1/content", func(r chi.Router) {
		// URL detection endpoint
		r.Post("/detect", detectionHandler.DetectURL)

		// Content management routes
		r.Post("/", contentHandler.CreateContent)
		r.Get("/{content_id}", contentHandler.GetContent)
		r.Put("/{content_id}", contentHandler.UpdateContent)

		// Bulk operation routes
		r.Post("/bulk", bulkHandler.BulkCreateContent)
		r.Post("/check-duplicate", bulkHandler.CheckDuplicates)

		// User-content routes (nested under /user/{user_id})
		r.Route("/user/{user_id}", func(r chi.Router) {
			r.Get("/", userContentHandler.ListUserContents)
			r.Post("/", userContentHandler.AddContentToUser)
			r.Get("/search", userContentHandler.SearchUserContents)
			r.Patch("/{content_id}", userContentHandler.UpdateUserContent)
			r.Delete("/{content_id}", userContentHandler.DeleteUserContent)
		})

		// Bulk user-content routes
		r.Post("/user/bulk", bulkHandler.BulkAddToUsers)
	})

	return r
}
