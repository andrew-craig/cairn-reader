// Package logging provides structured logging using the standard library log/slog package.
package logging

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger returns Gin middleware for structured request logging
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate or extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Create request-scoped logger
		reqLogger := logger.With(
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("client_ip", c.ClientIP()),
		)
		c.Set("logger", reqLogger)

		// Process request
		c.Next()

		// Log completed request
		duration := time.Since(start)
		status := c.Writer.Status()

		// Choose log level based on status code
		logFn := reqLogger.Info
		if status >= 500 {
			logFn = reqLogger.Error
		} else if status >= 400 {
			logFn = reqLogger.Warn
		}

		attrs := []any{
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.Int("size", c.Writer.Size()),
		}

		// Add user ID if authenticated
		if userID, exists := c.Get("user_id"); exists {
			attrs = append(attrs, slog.Any("user_id", userID))
		}

		logFn("http request completed", attrs...)
	}
}

// GetLogger retrieves the request-scoped logger from context
func GetLogger(c *gin.Context) *slog.Logger {
	if logger, exists := c.Get("logger"); exists {
		if l, ok := logger.(*slog.Logger); ok {
			return l
		}
	}
	return slog.Default()
}

// GetRequestID retrieves the request ID from the Gin context
// Returns the request ID string, or an empty string if not found
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
