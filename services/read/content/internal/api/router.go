package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/handlers"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/middleware"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates and configures the HTTP router
func NewRouter(db *database.DB, ingestRSSServiceURL string, authMiddleware *auth.Middleware, internalAuthMiddleware *auth.InternalAuthMiddleware) http.Handler {
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
	ingestRSSClient := service.NewIngestRSSClient(ingestRSSServiceURL)

	// Initialize handlers
	contentHandler := handlers.NewContentHandler(contentService)
	userContentHandler := handlers.NewUserContentHandler(userContentRepo, contentRepo, contentService, urlDetector, ingestRSSClient)
	bulkHandler := handlers.NewBulkHandler(contentService, userContentRepo, contentRepo)
	detectionHandler := handlers.NewDetectionHandler(urlDetector)
	subscriptionAggregator := handlers.NewSubscriptionAggregatorHandler(ingestRSSClient)

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
		fmt.Fprintf(w, `{"service":"content-service","version":"0.5.0","status":"Phase 2 - Router & Middleware Integration complete"}`)
	})

	// API v1 routes - all under /api/v1/content prefix for consistent service boundary
	r.Route("/api/v1/content", func(r chi.Router) {
		// URL detection endpoint (unprotected)
		r.Post("/detect", detectionHandler.DetectURL)

		// Content management routes (unprotected - used by internal services)
		r.Post("/", contentHandler.CreateContent)
		r.Get("/{content_id}", contentHandler.GetContent)
		r.Put("/{content_id}", contentHandler.UpdateContent)

		// Bulk operation routes (unprotected - used by internal services)
		r.Post("/bulk", bulkHandler.BulkCreateContent)
		r.Post("/check-duplicate", bulkHandler.CheckDuplicates)

		// Protected user-content routes (nested under /user/{user_id})
		r.Route("/user/{user_id}", func(r chi.Router) {
			// Apply RequireAuth middleware to all routes in this group
			r.Use(authMiddleware.RequireAuth)

			r.Get("/", userContentHandler.ListUserContents)
			r.Post("/", userContentHandler.AddContentToUser)
			r.Get("/search", userContentHandler.SearchUserContents)
			r.Patch("/{content_id}", userContentHandler.UpdateUserContent)
			r.Delete("/{content_id}", userContentHandler.DeleteUserContent)

			// Subscription aggregation endpoint - returns unified list from all sources
			r.Get("/subscriptions", subscriptionAggregator.ListAllSubscriptions)
		})

		// Protected bulk user-content route
		r.With(authMiddleware.RequireAuth).Post("/user/bulk", bulkHandler.BulkAddToUsers)
	})

	// Internal API routes - used by internal services (Ingest RSS, etc.)
	// Protected by API key-based service-to-service authentication
	r.Route("/api/v1/internal", func(r chi.Router) {
		r.Use(internalAuthMiddleware.RequireInternalAPIKey)
		// Internal bulk operations for Ingest RSS Service
		r.Post("/content/user/bulk", bulkHandler.BulkAddToUsersInternal)
	})

	return r
}
