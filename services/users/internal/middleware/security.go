package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RequireHTTPS enforces HTTPS connections in production environments
// It redirects HTTP requests to HTTPS or returns an error based on configuration
func RequireHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is already HTTPS
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check X-Forwarded-Proto header (set by reverse proxies)
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			next.ServeHTTP(w, r)
			return
		}

		// If not HTTPS, reject the request
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "HTTPS required",
		})
	})
}

// RequireHTTPSWithRedirect enforces HTTPS by redirecting HTTP requests
// This is useful in development/staging but should be used carefully in production
func RequireHTTPSWithRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is already HTTPS
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check X-Forwarded-Proto header (set by reverse proxies)
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			next.ServeHTTP(w, r)
			return
		}

		// Redirect to HTTPS
		httpsURL := "https://" + r.Host + r.RequestURI
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})
}

// SecureHeaders adds security-related HTTP headers
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking attacks
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable browser XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HSTS) - tells browsers to only use HTTPS
		// max-age=31536000 is 1 year
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy - helps prevent XSS attacks
		// This is a basic policy, adjust based on your needs
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Referrer Policy - controls how much referrer information is sent
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy - control which browser features can be used
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// SecureHeadersRelaxed adds security headers with more relaxed CSP for APIs
// that need to be consumed by various clients
func SecureHeadersRelaxed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking attacks
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable browser XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HSTS) - tells browsers to only use HTTPS
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// PreventCaching adds headers to prevent browser caching of sensitive responses
// Use this for endpoints that return sensitive data
func PreventCaching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

// AllowCaching adds headers to allow browser caching for specified duration
func AllowCaching(maxAge int) func(http.Handler) http.Handler {
	cacheControl := fmt.Sprintf("public, max-age=%d", maxAge)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", cacheControl)
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateContentType ensures requests have the correct Content-Type header
func ValidateContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for GET, HEAD, and OPTIONS requests
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			ct := r.Header.Get("Content-Type")
			// Remove charset and other parameters
			ct = strings.Split(ct, ";")[0]
			ct = strings.TrimSpace(ct)

			if ct != contentType {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid content type, expected " + contentType,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireJSON is a convenience middleware that requires application/json Content-Type
func RequireJSON() func(http.Handler) http.Handler {
	return ValidateContentType("application/json")
}
