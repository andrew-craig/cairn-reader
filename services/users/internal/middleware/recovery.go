package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/gin-gonic/gin"
)

// Recovery creates a middleware that recovers from panics and returns a 500 error
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID := logging.GetRequestID(c)
				if requestID == "" {
					requestID = "unknown"
				}

				// Get user ID if available
				userID := ""
				if id, e := auth.GetUserIDFromGinContext(c); e == nil {
					userID = id.String()
				}

				// Log the panic with stack trace
				slog.Error("panic recovered",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("user_id", userID),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.String("stack", string(debug.Stack())),
				)

				// Return 500 error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      "internal server error",
					"request_id": requestID,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// RecoveryWithDetails creates a middleware that recovers from panics and returns detailed error info
// WARNING: Only use in development! This exposes stack traces which can be a security risk
func RecoveryWithDetails() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID := logging.GetRequestID(c)
				if requestID == "" {
					requestID = "unknown"
				}

				// Get user ID if available
				userID := ""
				if id, e := auth.GetUserIDFromGinContext(c); e == nil {
					userID = id.String()
				}

				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic with stack trace
				slog.Error("panic recovered",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("user_id", userID),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.String("stack", stack),
				)

				// Return 500 error with details (development only!)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      fmt.Sprintf("%v", err),
					"request_id": requestID,
					"stack":      stack,
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

// RecoveryWithHandler creates a middleware that recovers from panics and calls a custom handler
type PanicHandler func(c *gin.Context, err interface{}, stack []byte)

func RecoveryWithHandler(handler PanicHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()

				// Call custom handler
				handler(c, err, stack)

				// Abort if not already aborted
				if !c.IsAborted() {
					c.Abort()
				}
			}
		}()

		c.Next()
	}
}

// DefaultPanicHandler is a default panic handler implementation
func DefaultPanicHandler(c *gin.Context, err interface{}, stack []byte) {
	requestID := logging.GetRequestID(c)
	if requestID == "" {
		requestID = "unknown"
	}

	userID := ""
	if id, e := auth.GetUserIDFromGinContext(c); e == nil {
		userID = id.String()
	}

	// Log the panic
	slog.Error("panic recovered",
		slog.String("request_id", requestID),
		slog.Any("error", err),
		slog.String("user_id", userID),
		slog.String("path", c.Request.URL.Path),
		slog.String("method", c.Request.Method),
		slog.String("client_ip", c.ClientIP()),
		slog.Time("timestamp", time.Now()),
		slog.String("stack", string(stack)),
	)

	// Return error response
	c.JSON(http.StatusInternalServerError, gin.H{
		"error":      "internal server error",
		"request_id": requestID,
	})
}

// RecoveryWithMetrics creates a middleware that recovers from panics and increments metrics
// This is useful for monitoring and alerting on panics in production
func RecoveryWithMetrics(metricsFunc func(path string, method string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Increment panic metric
				if metricsFunc != nil {
					metricsFunc(c.Request.URL.Path, c.Request.Method)
				}

				// Use default panic handler
				DefaultPanicHandler(c, err, debug.Stack())
				c.Abort()
			}
		}()

		c.Next()
	}
}

// SafeRecovery creates a production-ready recovery middleware that:
// - Logs panics with full context
// - Returns generic error messages (no stack traces)
// - Includes request ID for tracking
// - Doesn't expose internal details
func SafeRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := logging.GetRequestID(c)
				if requestID == "" {
					requestID = "unknown"
				}

				userID := ""
				if id, e := auth.GetUserIDFromGinContext(c); e == nil {
					userID = id.String()
				}

				// Log detailed panic information
				slog.Error("panic",
					slog.String("request_id", requestID),
					slog.Any("error", err),
					slog.String("user_id", userID),
					slog.String("client_ip", c.ClientIP()),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.Time("timestamp", time.Now()),
				)

				// Log stack trace separately (for analysis, not sent to client)
				slog.Error("stack trace",
					slog.String("request_id", requestID),
					slog.String("stack", string(debug.Stack())),
				)

				// Return generic error (don't expose internal details)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      "an unexpected error occurred, please try again later",
					"request_id": requestID,
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}
