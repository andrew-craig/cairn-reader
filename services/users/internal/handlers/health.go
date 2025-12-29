package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn/services/users/internal/auth"
	"github.com/andrew-craig/cairn/services/users/internal/database"
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

// HealthCheck returns a simple health check response
// GET /health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// ReadyCheck returns a detailed readiness check including database and dependencies
// GET /ready
func (h *HealthHandler) ReadyCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	// Check database health
	if err := h.db.Health(ctx); err != nil {
		checks["database"] = "unhealthy"
		allHealthy = false
	} else {
		checks["database"] = "healthy"
	}

	// Check Vault health
	if h.vaultClient != nil {
		if err := h.vaultClient.Health(); err != nil {
			checks["vault"] = "unhealthy"
			allHealthy = false
		} else {
			checks["vault"] = "healthy"
		}
	} else {
		checks["vault"] = "not_configured"
	}

	if allHealthy {
		c.JSON(http.StatusOK, HealthResponse{
			Status: "ready",
			Checks: checks,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:  "not_ready",
			Checks:  checks,
			Message: "One or more health checks failed",
		})
	}
}
