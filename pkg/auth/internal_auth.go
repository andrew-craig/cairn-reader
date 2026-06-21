package auth

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const (
	// InternalAPIKeyHeader is the header name for internal service-to-service authentication
	InternalAPIKeyHeader = "X-Internal-API-Key"
)

// InternalAuthMiddleware provides HTTP middleware for service-to-service API key authentication
type InternalAuthMiddleware struct {
	apiKey string
}

// NewInternalAuthMiddleware creates a new internal auth middleware with the given API key
func NewInternalAuthMiddleware(apiKey string) *InternalAuthMiddleware {
	return &InternalAuthMiddleware{
		apiKey: apiKey,
	}
}

// RequireInternalAPIKey is middleware that requires a valid internal API key.
// Returns 401 Unauthorized if the key is missing or invalid.
func (m *InternalAuthMiddleware) RequireInternalAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get(InternalAPIKeyHeader)
		if providedKey == "" {
			slog.Debug("internal auth: missing API key header")
			sendInternalAuthError(w, http.StatusUnauthorized, "unauthorized", "missing internal API key")
			return
		}

		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(m.apiKey)) != 1 {
			slog.Debug("internal auth: invalid API key")
			sendInternalAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid internal API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func sendInternalAuthError(w http.ResponseWriter, statusCode int, errCode string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errCode,
		Message: message,
		Meta: errorMeta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1",
		},
	})
}
