package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db          *database.DB
	vaultClient *auth.VaultClient
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.DB, vaultClient *auth.VaultClient) *HealthHandler {
	return &HealthHandler{
		db:          db,
		vaultClient: vaultClient,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
	Message string            `json:"message,omitempty"`
}

// LivenessCheck returns a simple liveness check response
// GET /health/live
// Used by orchestrators to determine if the service process should be restarted
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "healthy",
	})
}

// ReadinessCheck returns a detailed readiness check including database and dependencies
// GET /health/ready
// Returns 503 Service Unavailable if dependencies are unreachable
// Used by load balancers to determine if traffic should be routed to this instance
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
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

	if allHealthy {
		c.JSON(http.StatusOK, HealthResponse{
			Status: "healthy",
			Checks: checks,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:  "unhealthy",
			Checks:  checks,
			Message: "One or more dependencies are unavailable",
		})
	}
}
