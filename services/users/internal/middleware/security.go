package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequireHTTPS enforces HTTPS connections in production environments
// It redirects HTTP requests to HTTPS or returns an error based on configuration
func RequireHTTPS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the request is already HTTPS
		if c.Request.TLS != nil {
			c.Next()
			return
		}

		// Check X-Forwarded-Proto header (set by reverse proxies)
		if c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Next()
			return
		}

		// If not HTTPS, reject the request
		c.JSON(http.StatusForbidden, gin.H{
			"error": "HTTPS required",
		})
		c.Abort()
	}
}

// RequireHTTPSWithRedirect enforces HTTPS by redirecting HTTP requests
// This is useful in development/staging but should be used carefully in production
func RequireHTTPSWithRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the request is already HTTPS
		if c.Request.TLS != nil {
			c.Next()
			return
		}

		// Check X-Forwarded-Proto header (set by reverse proxies)
		if c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Next()
			return
		}

		// Redirect to HTTPS
		httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
		c.Redirect(http.StatusMovedPermanently, httpsURL)
		c.Abort()
	}
}

// SecureHeaders adds security-related HTTP headers
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable browser XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HSTS) - tells browsers to only use HTTPS
		// max-age=31536000 is 1 year
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy - helps prevent XSS attacks
		// This is a basic policy, adjust based on your needs
		c.Header("Content-Security-Policy", "default-src 'self'")

		// Referrer Policy - controls how much referrer information is sent
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy - control which browser features can be used
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// SecureHeadersRelaxed adds security headers with more relaxed CSP for APIs
// that need to be consumed by various clients
func SecureHeadersRelaxed() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable browser XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HSTS) - tells browsers to only use HTTPS
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// PreventCaching adds headers to prevent browser caching of sensitive responses
// Use this for endpoints that return sensitive data
func PreventCaching() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// AllowCaching adds headers to allow browser caching for specified duration
func AllowCaching(maxAge int) gin.HandlerFunc {
	cacheControl := "public, max-age=" + string(rune(maxAge))
	return func(c *gin.Context) {
		c.Header("Cache-Control", cacheControl)
		c.Next()
	}
}

// ValidateContentType ensures requests have the correct Content-Type header
func ValidateContentType(contentType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip validation for GET, HEAD, and OPTIONS requests
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		ct := c.GetHeader("Content-Type")
		// Remove charset and other parameters
		ct = strings.Split(ct, ";")[0]
		ct = strings.TrimSpace(ct)

		if ct != contentType {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "invalid content type, expected " + contentType,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireJSON is a convenience middleware that requires application/json Content-Type
func RequireJSON() gin.HandlerFunc {
	return ValidateContentType("application/json")
}
