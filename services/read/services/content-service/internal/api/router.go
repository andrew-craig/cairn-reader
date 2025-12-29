package api

import (
	"fmt"
	"net/http"

	"github.com/andrew-craig/cairn-read/services/content-service/internal/api/handlers"
	"github.com/andrew-craig/cairn-read/services/content-service/internal/api/middleware"
	"github.com/andrew-craig/cairn-read/services/content-service/internal/database"
	"github.com/andrew-craig/cairn-read/services/content-service/internal/repository"
	"github.com/andrew-craig/cairn-read/services/content-service/internal/service"
	"github.com/go-chi/chi/v5"
)

// NewRouter creates and configures the HTTP router
func NewRouter(db *database.DB) http.Handler {
	r := chi.NewRouter()

	// Apply global middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logging)
	r.Use(middleware.ValidateJSON)

	// Initialize repositories
	contentRepo := repository.NewContentRepository(db.DB)
	userContentRepo := repository.NewUserContentRepository(db.DB)

	// Initialize services
	contentService := service.NewContentService(contentRepo, db.DB)

	// Initialize handlers
	contentHandler := handlers.NewContentHandler(contentService)
	userContentHandler := handlers.NewUserContentHandler(userContentRepo, contentRepo)
	bulkHandler := handlers.NewBulkHandler(contentService, userContentRepo, contentRepo)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		ctx := r.Context()
		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"error","service":"content-service","message":"Database connection failed","error":"%s"}`, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"content-service","message":"Phase 1.4: Bulk Operations complete"}`)
	})

	// Root endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service":"content-service","version":"0.4.0","status":"Phase 1.4 - Bulk Operations complete"}`)
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Content routes
		r.Post("/contents", contentHandler.CreateContent)
		r.Get("/contents/{id}", contentHandler.GetContent)
		r.Put("/contents/{id}", contentHandler.UpdateContent)

		// Bulk operation routes
		r.Post("/contents/bulk", bulkHandler.BulkCreateContent)
		r.Post("/contents/check-duplicates", bulkHandler.CheckDuplicates)

		// User-content routes
		r.Route("/users/{user_id}/contents", func(r chi.Router) {
			r.Get("/", userContentHandler.ListUserContents)
			r.Post("/", userContentHandler.AddContentToUser)
			r.Get("/search", userContentHandler.SearchUserContents)
			r.Patch("/{content_id}", userContentHandler.UpdateUserContent)
			r.Delete("/{content_id}", userContentHandler.DeleteUserContent)
		})

		// Bulk user-content routes
		r.Post("/users/bulk/contents", bulkHandler.BulkAddToUsers)
	})

	return r
}
