// Package middleware provides HTTP middleware for the email ingest service.
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
)

const (
	// APIKeyHeader is the header name for API key authentication
	APIKeyHeader = "X-API-Key"
)

// APIKeyAuth provides API key authentication middleware for the ingest endpoint.
// Validates X-API-Key header against hashed keys in the api_keys table.
type APIKeyAuth struct {
	repo repository.APIKeyRepository
}

// NewAPIKeyAuth creates a new API key authentication middleware
func NewAPIKeyAuth(repo repository.APIKeyRepository) *APIKeyAuth {
	return &APIKeyAuth{repo: repo}
}

// RequireAPIKey is middleware that requires a valid API key in the X-API-Key header.
// Returns 401 Unauthorized if the key is missing or invalid.
func (m *APIKeyAuth) RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(APIKeyHeader)
		if apiKey == "" {
			slog.Debug("api key auth: missing API key header")
			sendAPIKeyError(w, http.StatusUnauthorized, "unauthorized", "missing API key")
			return
		}

		valid, err := m.repo.ValidateKey(r.Context(), apiKey)
		if err != nil {
			slog.Error("api key auth: validation error", "error", err.Error())
			sendAPIKeyError(w, http.StatusInternalServerError, "internal_error", "failed to validate API key")
			return
		}

		if !valid {
			slog.Debug("api key auth: invalid API key")
			sendAPIKeyError(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// errorResponse represents a JSON error response
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func sendAPIKeyError(w http.ResponseWriter, statusCode int, errCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{
		Error:   errCode,
		Message: message,
	})
}

// MockAPIKeyRepository is a mock for testing
type MockAPIKeyRepository struct {
	ValidateKeyFn     func(ctx context.Context, apiKey string) (bool, error)
	GetKeyHashByNameFn func(ctx context.Context, keyName string) (string, error)
}

func (m *MockAPIKeyRepository) ValidateKey(ctx context.Context, apiKey string) (bool, error) {
	return m.ValidateKeyFn(ctx, apiKey)
}

func (m *MockAPIKeyRepository) GetKeyHashByName(ctx context.Context, keyName string) (string, error) {
	if m.GetKeyHashByNameFn != nil {
		return m.GetKeyHashByNameFn(ctx, keyName)
	}
	return "", nil
}
