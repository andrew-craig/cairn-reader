package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
)

// Recovery creates a middleware that recovers from panics and returns a 500 error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID := logging.GetRequestIDFromContext(r.Context())
				if requestID == "" {
					requestID = "unknown"
				}

				// Get user ID if available
				userID := ""
				if id, e := auth.GetUserIDOrError(r.Context()); e == nil {
					userID = id.String()
				}

				// Log the panic with stack trace
				slog.Error("panic recovered",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("user_id", userID),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("stack", string(debug.Stack())),
				)

				// Return 500 error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error":      "internal server error",
					"request_id": requestID,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RecoveryWithDetails creates a middleware that recovers from panics and returns detailed error info
// WARNING: Only use in development! This exposes stack traces which can be a security risk
func RecoveryWithDetails(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID := logging.GetRequestIDFromContext(r.Context())
				if requestID == "" {
					requestID = "unknown"
				}

				// Get user ID if available
				userID := ""
				if id, e := auth.GetUserIDOrError(r.Context()); e == nil {
					userID = id.String()
				}

				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic with stack trace
				slog.Error("panic recovered",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("user_id", userID),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("stack", stack),
				)

				// Return 500 error with details (development only!)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      fmt.Sprintf("%v", err),
					"request_id": requestID,
					"stack":      stack,
					"path":       r.URL.Path,
					"method":     r.Method,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// PanicHandler is a function type for custom panic handlers
type PanicHandler func(w http.ResponseWriter, r *http.Request, err interface{}, stack []byte)

// RecoveryWithHandler creates a middleware that recovers from panics and calls a custom handler
func RecoveryWithHandler(handler PanicHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()

					// Call custom handler
					handler(w, r, err, stack)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultPanicHandler is a default panic handler implementation
func DefaultPanicHandler(w http.ResponseWriter, r *http.Request, err interface{}, stack []byte) {
	requestID := logging.GetRequestIDFromContext(r.Context())
	if requestID == "" {
		requestID = "unknown"
	}

	userID := ""
	if id, e := auth.GetUserIDOrError(r.Context()); e == nil {
		userID = id.String()
	}

	// Log the panic
	slog.Error("panic recovered",
		slog.String("request_id", requestID),
		slog.Any("error", err),
		slog.String("user_id", userID),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.String("client_ip", getClientIP(r)),
		slog.Time("timestamp", time.Now()),
		slog.String("stack", string(stack)),
	)

	// Return error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"error":      "internal server error",
		"request_id": requestID,
	})
}

// RecoveryWithMetrics creates a middleware that recovers from panics and increments metrics
// This is useful for monitoring and alerting on panics in production
func RecoveryWithMetrics(metricsFunc func(path string, method string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Increment panic metric
					if metricsFunc != nil {
						metricsFunc(r.URL.Path, r.Method)
					}

					// Use default panic handler
					DefaultPanicHandler(w, r, err, debug.Stack())
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// SafeRecovery creates a production-ready recovery middleware that:
// - Logs panics with full context
// - Returns generic error messages (no stack traces)
// - Includes request ID for tracking
// - Doesn't expose internal details
func SafeRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := logging.GetRequestIDFromContext(r.Context())
				if requestID == "" {
					requestID = "unknown"
				}

				userID := ""
				if id, e := auth.GetUserIDOrError(r.Context()); e == nil {
					userID = id.String()
				}

				// Log detailed panic information
				slog.Error("panic",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("user_id", userID),
					slog.String("client_ip", getClientIP(r)),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.Time("timestamp", time.Now()),
				)

				// Log stack trace separately (for analysis, not sent to client)
				slog.Error("stack trace",
					slog.String("request_id", requestID),
					slog.String("stack", string(debug.Stack())),
				)

				// Return generic error (don't expose internal details)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error":      "an unexpected error occurred, please try again later",
					"request_id": requestID,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
