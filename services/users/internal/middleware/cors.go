package middleware

import (
	"net/http"
	"strconv"
	"strings"
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

// CORS creates a middleware with custom CORS configuration
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

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
						w.Header().Set("Access-Control-Allow-Origin", "*")
					} else {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
					}

					// Set Access-Control-Allow-Credentials
					if config.AllowCredentials {
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}

					// Set Access-Control-Expose-Headers
					if len(config.ExposeHeaders) > 0 {
						w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
					}
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				// Set Access-Control-Allow-Methods
				if len(config.AllowMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
				}

				// Set Access-Control-Allow-Headers
				if len(config.AllowHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
				}

				// Set Access-Control-Max-Age
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSForOrigins creates a CORS middleware for specific origins
func CORSForOrigins(origins ...string) func(http.Handler) http.Handler {
	config := StrictCORSConfig(origins)
	return CORS(config)
}

// DevelopmentCORS creates a permissive CORS middleware for development
// WARNING: Do not use in production!
func DevelopmentCORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "43200")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ProductionCORS creates a strict CORS middleware for production
func ProductionCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		// Default to no origins if none specified
		allowedOrigins = []string{}
	}

	config := StrictCORSConfig(allowedOrigins)
	return CORS(config)
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
func CORSWithWildcard(allowedOrigins []string) func(http.Handler) http.Handler {
	config := StrictCORSConfig(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && isOriginAllowed(origin, config.AllowOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if len(config.ExposeHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				if len(config.AllowMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
				}

				if len(config.AllowHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
				}

				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
