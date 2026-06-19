package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ContextKey is the type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDContextKey is the context key for storing the user ID
	UserIDContextKey ContextKey = "user_id"
)

// Errors
var (
	// ErrUserIDNotFound is returned when user ID is not found in context
	ErrUserIDNotFound = errors.New("user ID not found in context - ensure RequireAuth middleware is applied")
)

// errorMeta is the metadata included in error responses
type errorMeta struct {
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// ErrorResponse represents a JSON error response matching the standard pkg/api envelope
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
	Meta    errorMeta         `json:"meta"`
}

// Middleware provides HTTP middleware for JWT authentication
type Middleware struct {
	validator *Validator
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(validator *Validator) *Middleware {
	return &Middleware{
		validator: validator,
	}
}

// RequireAuth is middleware that requires a valid JWT token
// If the token is invalid or missing, it returns a 401 Unauthorized response
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		token, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			slog.Debug("auth: token extraction failed", "error", err.Error())
			m.sendError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing authentication token")
			return
		}

		// Validate token
		claims, err := m.validator.ValidateToken(token)
		if err != nil {
			slog.Debug("auth: token validation failed", "error", err.Error())
			m.sendError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing authentication token")
			return
		}

		// Store user ID in context
		ctx := SetUserIDInContext(r.Context(), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is middleware for endpoints where credentials are optional —
// authenticated callers receive a personalized response, anonymous callers a
// public one. Use IsAuthenticated() in the handler to branch on auth state.
//
// Semantics (fail-closed when credentials are presented):
//   - No Authorization header: continue unauthenticated.
//   - Authorization header present but malformed or carrying an invalid token
//     (expired, bad signature, wrong issuer/audience, etc.): respond 401.
//
// The "optional" part is whether credentials are presented, not whether they
// are valid. Silently dropping an invalid token would hide token-abuse
// attempts and let a stale session pretend to be anonymous.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		token, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			slog.Warn("auth: rejected request on optional-auth endpoint: malformed Authorization header",
				"error", err.Error(),
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			m.sendError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing authentication token")
			return
		}

		claims, err := m.validator.ValidateToken(token)
		if err != nil {
			slog.Warn("auth: rejected request on optional-auth endpoint: token validation failed",
				"error", err.Error(),
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			m.sendError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing authentication token")
			return
		}

		ctx := SetUserIDInContext(r.Context(), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sendError sends a JSON error response
func (m *Middleware) sendError(w http.ResponseWriter, statusCode int, error string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   error,
		Message: message,
		Meta: errorMeta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   "v1",
		},
	})
}

// SetUserIDInContext adds the user ID to the request context
// Returns a new context with the user ID value
func SetUserIDInContext(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDContextKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the request context
// Returns the user ID and true if authenticated, or zero UUID and false if not
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return userID, ok
}

// GetUserIDOrError retrieves the user ID from context and returns an error if not found
// This is the recommended approach for handlers that require authentication.
// Returns ErrUserIDNotFound if user ID is not in context.
func GetUserIDOrError(ctx context.Context) (uuid.UUID, error) {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		return uuid.Nil, ErrUserIDNotFound
	}
	return userID, nil
}

// MustGetUserID retrieves the user ID from context and panics if not found
// DEPRECATED: Use GetUserIDOrError instead to avoid service crashes on programming errors.
// Use this only in handlers that are protected by RequireAuth middleware.
func MustGetUserID(ctx context.Context) uuid.UUID {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		panic("user ID not found in context - ensure RequireAuth middleware is applied")
	}
	return userID
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetUserIDFromContext(ctx)
	return ok
}

// MiddlewareFunc is a function type for creating middleware
type MiddlewareFunc func(http.Handler) http.Handler

// Chain chains multiple middleware functions together
func Chain(middlewares ...MiddlewareFunc) MiddlewareFunc {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
