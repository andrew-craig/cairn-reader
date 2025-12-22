package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// Recovery creates a middleware that recovers from panics and returns a 500 error
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID if available
				requestID := GetRequestID(c)
				if requestID == "" {
					requestID = "unknown"
				}

				// Get user ID if available
				userID := ""
				if id, e := GetUserIDFromContext(c); e == nil {
					userID = id.String()
				}

				// Log the panic with stack trace
				log.Printf(
					"PANIC RECOVERED: [%s] error=%v user=%s path=%s method=%s stack=%s",
					requestID,
					err,
					userID,
					c.Request.URL.Path,
					c.Request.Method,
					debug.Stack(),
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
				requestID := GetRequestID(c)
				if requestID == "" {
					requestID = "unknown"
				}

				// Get user ID if available
				userID := ""
				if id, e := GetUserIDFromContext(c); e == nil {
					userID = id.String()
				}

				// Get stack trace
				stack := string(debug.Stack())

				// Log the panic with stack trace
				log.Printf(
					"PANIC RECOVERED: [%s] error=%v user=%s path=%s method=%s\nstack:\n%s",
					requestID,
					err,
					userID,
					c.Request.URL.Path,
					c.Request.Method,
					stack,
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
	requestID := GetRequestID(c)
	if requestID == "" {
		requestID = "unknown"
	}

	userID := ""
	if id, e := GetUserIDFromContext(c); e == nil {
		userID = id.String()
	}

	// Log the panic
	log.Printf(
		"PANIC RECOVERED: [%s] error=%v user=%s path=%s method=%s ip=%s timestamp=%s\nstack:\n%s",
		requestID,
		err,
		userID,
		c.Request.URL.Path,
		c.Request.Method,
		c.ClientIP(),
		time.Now().Format(time.RFC3339),
		stack,
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
				requestID := GetRequestID(c)
				if requestID == "" {
					requestID = "unknown"
				}

				userID := ""
				if id, e := GetUserIDFromContext(c); e == nil {
					userID = id.String()
				}

				// Log detailed panic information
				log.Printf(
					"PANIC: [%s] error=%v user=%s ip=%s path=%s method=%s timestamp=%s",
					requestID,
					err,
					userID,
					c.ClientIP(),
					c.Request.URL.Path,
					c.Request.Method,
					time.Now().Format(time.RFC3339),
				)

				// Log stack trace separately (for analysis, not sent to client)
				log.Printf("Stack trace for request [%s]:\n%s", requestID, debug.Stack())

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
