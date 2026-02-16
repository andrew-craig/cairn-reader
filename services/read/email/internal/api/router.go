// Package api provides HTTP routing and handlers for the email ingest service.
package api

import (
	"net/http"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the HTTP router
func NewRouter(db *database.DB) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health checks
	r.Get("/health/live", handleLiveness)
	r.Get("/health/ready", handleReadiness(db))

	// API routes (to be implemented)
	r.Route("/api/v1/source/email", func(r chi.Router) {
		// Email address management (JWT protected)
		// POST   /user/{user_id}/address
		// GET    /user/{user_id}/address
		// DELETE /user/{user_id}/address

		// Email ingestion (API key protected)
		// POST /ingest

		// Sender management (JWT protected)
		// GET /user/{user_id}/senders
	})

	return r
}

// handleLiveness returns a simple liveness probe response
func handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReadiness returns a readiness probe that checks database connectivity
func handleReadiness(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "Database not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
