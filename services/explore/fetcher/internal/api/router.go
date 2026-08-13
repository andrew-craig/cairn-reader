// Package api provides the HTTP router for the explore fetcher service:
// health checks and the manual fetch/sync/stats trigger endpoints.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	pkgapi "github.com/cairn-app/cairn-reader/pkg/api"
	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/sync"
	"github.com/go-chi/chi/v5"
)

// Pinger is satisfied by *pgxpool.Pool; narrowed here so the readiness check
// can be exercised without a real database connection in tests.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewRouter creates and configures the HTTP router for the explore fetcher service.
// bgCtx is the long-lived context background triggers run under (cancelled on shutdown).
func NewRouter(bgCtx context.Context, database Pinger, feedRepo db.FeedRepositoryInterface, feedFetcher *fetcher.Fetcher, feedSyncer *sync.FeedSyncer, internalAuthMiddleware *auth.InternalAuthMiddleware, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(sharedmw.Recovery)
	r.Use(logging.ChiRequestLogger(logger))
	r.Use(sharedmw.SecureHeadersRelaxed)

	// Health check endpoints
	r.Get("/health/live", livenessHandler)
	r.Head("/health/live", livenessHandler)
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessHandler(w, r, database)
	})
	r.Head("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		readinessHandler(w, r, database)
	})

	// API v1 routes - manual operator triggers, internal service-to-service only.
	r.Route("/api/v1/explore/feed", func(r chi.Router) {
		r.Use(sharedmw.RequireHTTPS)
		r.Use(internalAuthMiddleware.RequireInternalAPIKey)

		r.Post("/fetch", func(w http.ResponseWriter, r *http.Request) {
			go func() {
				if err := feedFetcher.FetchSingleFeed(bgCtx); err != nil {
					slog.Error("error in fetch goroutine", slog.Any("error", err))
				}
			}()
			pkgapi.WriteSuccess(w, http.StatusAccepted, map[string]string{"status": "fetch triggered"}, "v1")
		})

		r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			total, enabled, disabled, neverFetched, err := feedRepo.GetFeedStats(r.Context())
			if err != nil {
				pkgapi.WriteError(w, http.StatusInternalServerError, pkgapi.ErrCodeInternal, "Failed to retrieve feed statistics", nil, "v1")
				return
			}
			stats := map[string]int{
				"total":         total,
				"enabled":       enabled,
				"disabled":      disabled,
				"never_fetched": neverFetched,
			}
			pkgapi.WriteSuccess(w, http.StatusOK, stats, "v1")
		})

		r.Post("/sync", func(w http.ResponseWriter, r *http.Request) {
			go func() {
				if err := feedSyncer.SyncOnce(bgCtx); err != nil {
					slog.Error("error in sync goroutine", slog.Any("error", err))
				}
			}()
			pkgapi.WriteSuccess(w, http.StatusAccepted, map[string]string{"status": "sync triggered"}, "v1")
		})
	})

	return r
}

// livenessHandler returns a simple liveness check (process is running)
func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := struct {
		Status    string `json:"status"`
		Timestamp string `json:"timestamp"`
	}{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("error encoding response", "error", err)
	}
}

// readinessHandler checks if the service is ready to accept traffic.
// Returns 503 Service Unavailable if dependencies are unreachable.
func readinessHandler(w http.ResponseWriter, r *http.Request, database Pinger) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	status := "healthy"
	statusCode := http.StatusOK

	if err := database.Ping(ctx); err != nil {
		checks["database"] = "error"
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
		slog.Warn("database health check failed", slog.Any("error", err))
	} else {
		checks["database"] = "ok"
	}

	response := struct {
		Status    string            `json:"status"`
		Timestamp string            `json:"timestamp"`
		Checks    map[string]string `json:"checks"`
	}{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("error encoding response", "error", err)
	}
}
