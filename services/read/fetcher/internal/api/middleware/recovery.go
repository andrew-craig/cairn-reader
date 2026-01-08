package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery is a middleware that recovers from panics and logs the error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic and stack trace
				slog.Error("PANIC recovered", "panic", err, "stack", string(debug.Stack()))

				// Return 500 Internal Server Error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "internal_error",
					"message": "An internal server error occurred",
					"code":    http.StatusInternalServerError,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
