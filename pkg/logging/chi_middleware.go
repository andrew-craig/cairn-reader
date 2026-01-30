// Package logging provides structured logging using the standard library log/slog package.
package logging

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// ChiRequestLogger returns Chi middleware for structured request logging
func ChiRequestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate or extract request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			w.Header().Set("X-Request-ID", requestID)

			// Create request-scoped logger
			reqLogger := logger.With(
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("client_ip", r.RemoteAddr),
			)

			// Wrap the response writer to capture status code and size
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Store logger and request ID in context
			ctx := WithLogger(r.Context(), reqLogger)
			ctx = WithRequestID(ctx, requestID)
			r = r.WithContext(ctx)

			// Process request
			next.ServeHTTP(ww, r)

			// Log completed request
			duration := time.Since(start)
			status := ww.Status()

			// Choose log level based on status code
			logFn := reqLogger.Info
			if status >= 500 {
				logFn = reqLogger.Error
			} else if status >= 400 {
				logFn = reqLogger.Warn
			}

			logFn("http request completed",
				slog.Int("status", status),
				slog.Duration("duration", duration),
				slog.Int("size", ww.BytesWritten()),
			)
		})
	}
}
