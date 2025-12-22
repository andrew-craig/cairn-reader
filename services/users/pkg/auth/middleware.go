package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// ContextKey is the type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDContextKey is the context key for storing the user ID
	UserIDContextKey ContextKey = "user_id"
)

// ErrorResponse represents a JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
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
			m.sendError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}

		// Validate token
		claims, err := m.validator.ValidateToken(token)
		if err != nil {
			m.sendError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}

		// Store user ID in context
		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is middleware that validates JWT token if present, but allows
// request to continue if token is missing. Use IsAuthenticated() to check auth status.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		token, err := ExtractTokenFromHeader(authHeader)
		if err != nil {
			// No token or invalid format - continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Validate token
		claims, err := m.validator.ValidateToken(token)
		if err != nil {
			// Invalid token - continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Store user ID in context
		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
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
	})
}

// GetUserIDFromContext retrieves the user ID from the request context
// Returns the user ID and true if authenticated, or zero UUID and false if not
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return userID, ok
}

// MustGetUserID retrieves the user ID from context and panics if not found
// Use this only in handlers that are protected by RequireAuth middleware
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
