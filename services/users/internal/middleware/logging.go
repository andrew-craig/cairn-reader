package middleware

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/andrew-craig/cairn/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger logs HTTP requests with timing and status information
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Get path before processing
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get user ID if authenticated
		userID := ""
		if id, err := auth.GetUserIDFromGinContext(c); err == nil {
			userID = id.String()
		}

		// Build query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Log the request
		log.Printf(
			"[%s] %s %s %d %v %s user=%s",
			requestID,
			c.Request.Method,
			path,
			statusCode,
			duration,
			clientIP,
			userID,
		)
	}
}

// RequestLoggerDetailed logs HTTP requests with more detailed information
// including request and response bodies (be careful with sensitive data)
func RequestLoggerDetailed() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Read request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// Restore the request body so it can be read again by handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get user ID if authenticated
		userID := ""
		if id, err := auth.GetUserIDFromGinContext(c); err == nil {
			userID = id.String()
		}

		// Log the request
		log.Printf(
			"[%s] %s %s %d %v %s user=%s req_size=%d resp_size=%d",
			requestID,
			c.Request.Method,
			c.Request.URL.Path,
			statusCode,
			duration,
			clientIP,
			userID,
			len(requestBody),
			writer.body.Len(),
		)
	}
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// RequestID generates or retrieves a request ID for tracking
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in headers
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in context and add to response
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// AuthenticationLogger specifically logs authentication attempts
func AuthenticationLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Determine success/failure
		success := statusCode >= 200 && statusCode < 300

		// Get user ID if available (might be in response or context)
		userID := ""
		if id, err := auth.GetUserIDFromGinContext(c); err == nil {
			userID = id.String()
		}

		// Log authentication attempt
		if success {
			log.Printf(
				"AUTH_SUCCESS: method=%s path=%s ip=%s user=%s duration=%v",
				c.Request.Method,
				path,
				clientIP,
				userID,
				time.Since(start),
			)
		} else {
			log.Printf(
				"AUTH_FAILURE: method=%s path=%s ip=%s status=%d duration=%v",
				c.Request.Method,
				path,
				clientIP,
				statusCode,
				time.Since(start),
			)
		}
	}
}

// SensitiveEndpointLogger logs access to sensitive endpoints
// This should be used for operations like account deletion, password changes, etc.
func SensitiveEndpointLogger(operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get user ID
		userID := ""
		if id, err := auth.GetUserIDFromGinContext(c); err == nil {
			userID = id.String()
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Get request ID
		requestID := GetRequestID(c)

		// Log before processing
		log.Printf(
			"SENSITIVE_OP_START: [%s] operation=%s user=%s ip=%s",
			requestID,
			operation,
			userID,
			clientIP,
		)

		// Process request
		c.Next()

		// Get status code
		statusCode := c.Writer.Status()
		success := statusCode >= 200 && statusCode < 300

		// Log after processing
		log.Printf(
			"SENSITIVE_OP_END: [%s] operation=%s user=%s ip=%s status=%d success=%v duration=%v",
			requestID,
			operation,
			userID,
			clientIP,
			statusCode,
			success,
			time.Since(start),
		)
	}
}
