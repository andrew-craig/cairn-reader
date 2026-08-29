// Package middleware provides shared HTTP middleware components for all Cairn services.
// This includes panic recovery, CORS, security headers, and rate limiting.
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/logging"
)

// Recovery creates a production-safe recovery middleware that:
//   - Logs panics with full context (request ID, path, method, client IP, stack trace)
//   - Returns a generic JSON error with request ID for tracking
//   - Never exposes internal details to clients
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := logging.GetRequestIDFromContext(r.Context())
				if requestID == "" {
					requestID = "unknown"
				}

				slog.Error("panic",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("client_ip", getClientIP(r)),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
				)

				slog.Error("stack trace",
					slog.String("request_id", requestID),
					slog.String("stack", string(debug.Stack())),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "internal_error",
					"message": "an unexpected error occurred, please try again later",
					"details": map[string]string{"request_id": requestID},
					"meta": map[string]string{
						"timestamp": time.Now().UTC().Format(time.RFC3339),
						"version":   "v1",
					},
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP address from the request,
// checking X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
