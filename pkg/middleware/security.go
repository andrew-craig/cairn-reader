package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RequireHTTPS enforces HTTPS connections by checking TLS state
// and the X-Forwarded-Proto header (set by reverse proxies).
func RequireHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("X-Forwarded-Proto") == "https" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "HTTPS required",
		})
	})
}

// RequireHTTPSWithRedirect enforces HTTPS by redirecting HTTP requests to HTTPS.
// Useful in staging environments; use RequireHTTPS for strict enforcement in production.
func RequireHTTPSWithRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("X-Forwarded-Proto") == "https" {
			next.ServeHTTP(w, r)
			return
		}

		httpsURL := "https://" + r.Host + r.RequestURI
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})
}

// SecureHeaders adds comprehensive security headers including
// X-Frame-Options, X-Content-Type-Options, HSTS, CSP, Referrer-Policy,
// and Permissions-Policy.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

// SecureHeadersRelaxed adds security headers with a more relaxed CSP,
// suitable for APIs consumed by various clients.
func SecureHeadersRelaxed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// PreventCaching adds headers to prevent browser caching of sensitive responses.
func PreventCaching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

// AllowCaching adds headers to allow browser caching for the specified duration in seconds.
func AllowCaching(maxAge int) func(http.Handler) http.Handler {
	cacheControl := fmt.Sprintf("public, max-age=%d", maxAge)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", cacheControl)
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateContentType ensures requests have the correct Content-Type header.
// GET, HEAD, and OPTIONS requests are skipped.
func ValidateContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			ct := r.Header.Get("Content-Type")
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

// RequireJSON is a convenience middleware that requires application/json Content-Type.
func RequireJSON() func(http.Handler) http.Handler {
	return ValidateContentType("application/json")
}
