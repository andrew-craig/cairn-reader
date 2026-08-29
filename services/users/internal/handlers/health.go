package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn-reader/services/users/internal/auth"
	"github.com/andrew-craig/cairn-reader/services/users/internal/database"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db          *database.DB
	vaultClient *auth.VaultClient
	version     string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB, vaultClient *auth.VaultClient, version string) *HealthHandler {
	return &HealthHandler{
		db:          db,
		vaultClient: vaultClient,
		version:     version,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version,omitempty"`
	Checks  map[string]string `json:"checks,omitempty"`
	Message string            `json:"message,omitempty"`
}

// LivenessCheck returns a simple liveness check response
// GET /health/live
// Used by orchestrators to determine if the service process should be restarted
func (h *HealthHandler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:  "healthy",
		Version: h.version,
	})
}

// ReadinessCheck returns a detailed readiness check including database and dependencies
// GET /health/ready
// Returns 503 Service Unavailable if dependencies are unreachable
// Used by load balancers to determine if traffic should be routed to this instance
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	// Check database health
	if err := h.db.Health(ctx); err != nil {
		checks["database"] = "error"
		allHealthy = false
	} else {
		checks["database"] = "ok"
	}

	// Check Vault health
	if h.vaultClient != nil {
		if err := h.vaultClient.Health(); err != nil {
			checks["vault"] = "error"
			allHealthy = false
		} else {
			checks["vault"] = "ok"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:  "healthy",
			Version: h.version,
			Checks:  checks,
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status:  "unhealthy",
			Checks:  checks,
			Message: "One or more dependencies are unavailable",
		})
	}
}
