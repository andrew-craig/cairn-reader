// Package logging provides structured logging using the standard library log/slog package.
package logging

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	loggerKey    contextKey = "logger"
	requestIDKey contextKey = "request_id"

	// HeaderXRequestID is the canonical header name for cross-service request tracing.
	HeaderXRequestID = "X-Request-ID"
)

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context, or returns default
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestIDFromContext retrieves the request ID from context, or returns empty string
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// SetRequestIDHeader copies the X-Request-ID from ctx onto req.
// If ctx contains no request ID a new UUID is generated and used.
// Call this before executing any outbound service-to-service HTTP request.
func SetRequestIDHeader(ctx context.Context, req *http.Request) {
	id := GetRequestIDFromContext(ctx)
	if id == "" {
		id = uuid.New().String()
	}
	req.Header.Set(HeaderXRequestID, id)
}
