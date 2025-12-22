package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           43200, // 12 hours
	}
}

// StrictCORSConfig returns a strict CORS configuration for production
// You should customize AllowOrigins for your specific domains
func StrictCORSConfig(allowedOrigins []string) CORSConfig {
	return CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           43200, // 12 hours
	}
}

// CORS creates a middleware with default CORS configuration
func CORS() gin.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig creates a middleware with custom CORS configuration
func CORSWithConfig(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" {
			allowed := false
			for _, allowedOrigin := range config.AllowOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				// Set Access-Control-Allow-Origin
				if len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*" {
					c.Header("Access-Control-Allow-Origin", "*")
				} else {
					c.Header("Access-Control-Allow-Origin", origin)
					c.Header("Vary", "Origin")
				}

				// Set Access-Control-Allow-Credentials
				if config.AllowCredentials {
					c.Header("Access-Control-Allow-Credentials", "true")
				}

				// Set Access-Control-Expose-Headers
				if len(config.ExposeHeaders) > 0 {
					c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
				}
			}
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			// Set Access-Control-Allow-Methods
			if len(config.AllowMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			}

			// Set Access-Control-Allow-Headers
			if len(config.AllowHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			}

			// Set Access-Control-Max-Age
			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSForOrigins creates a CORS middleware for specific origins
func CORSForOrigins(origins ...string) gin.HandlerFunc {
	config := StrictCORSConfig(origins)
	return CORSWithConfig(config)
}

// DevelopmentCORS creates a permissive CORS middleware for development
// WARNING: Do not use in production!
func DevelopmentCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "43200")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ProductionCORS creates a strict CORS middleware for production
func ProductionCORS(allowedOrigins []string) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		// Default to no origins if none specified
		allowedOrigins = []string{}
	}

	config := StrictCORSConfig(allowedOrigins)
	return CORSWithConfig(config)
}

// isOriginAllowed checks if an origin is allowed based on the configuration
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Support wildcard subdomains like *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}

// CORSWithWildcard creates a CORS middleware that supports wildcard subdomains
func CORSWithWildcard(allowedOrigins []string) gin.HandlerFunc {
	config := StrictCORSConfig(allowedOrigins)

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if origin != "" && isOriginAllowed(origin, config.AllowOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")

			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			if len(config.ExposeHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			}
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			if len(config.AllowMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			}

			if len(config.AllowHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			}

			if config.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
