// Package api provides HTTP routing and handlers for the email ingest service.
package api

import (
	"net/http"

	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/handlers"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/api/middleware"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/database"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// RouterDeps holds all dependencies for the HTTP router.
type RouterDeps struct {
	DB             *database.DB
	IngestHandler  *handlers.IngestHandler
	AddressHandler *handlers.AddressHandler
	SenderHandler  *handlers.SenderHandler
	APIKeyAuth     *middleware.APIKeyAuth
	JWTAuth        *middleware.JWTAuth
}

// NewRouter creates and configures the HTTP router
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(sharedmw.Recovery)
	r.Use(sharedmw.SecureHeadersRelaxed)

	// Health checks
	r.Get("/health/live", handleLiveness)
	r.Get("/health/ready", handleReadiness(deps.DB))

	// API routes
	r.Route("/api/v1/source/email", func(r chi.Router) {
		r.Use(sharedmw.RequireHTTPS)
		// Email ingestion (API key protected)
		r.With(deps.APIKeyAuth.RequireAPIKey).Post("/ingest", deps.IngestHandler.IngestEmail)

		// User-facing endpoints (JWT protected)
		r.Group(func(r chi.Router) {
			r.Use(deps.JWTAuth.RequireAuth)

			// Email address management
			r.Post("/user/{user_id}/address", deps.AddressHandler.GetOrCreate)
			r.Get("/user/{user_id}/address", deps.AddressHandler.Get)

			// Sender management
			r.Get("/user/{user_id}/senders", deps.SenderHandler.ListSenders)
		})
	})

	return r
}

// handleLiveness returns a simple liveness probe response
func handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// handleReadiness returns a readiness probe that checks database connectivity
func handleReadiness(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			http.Error(w, "Database not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}
