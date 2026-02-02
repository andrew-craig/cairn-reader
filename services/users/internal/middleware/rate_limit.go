// Package middleware provides HTTP middleware components for the user service.
// This file implements a token bucket rate limiter that supports per-IP,
// per-user, and custom key-based rate limiting with automatic cleanup
// of stale entries to prevent memory leaks.
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	requests map[string]*bucket
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	cleanup  time.Duration
}

// bucket represents a token bucket for a specific client
type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// limit: maximum number of requests allowed
// window: time window for the limit (e.g., 1 minute)
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*bucket),
		limit:    limit,
		window:   window,
		cleanup:  window * 2, // Cleanup old entries after 2x the window
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes old entries to prevent memory leaks
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.requests {
			b.mu.Lock()
			if now.Sub(b.lastRefill) > rl.cleanup {
				delete(rl.requests, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// allow checks if a request should be allowed
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.RLock()
	b, exists := rl.requests[key]
	rl.mu.RUnlock()

	if !exists {
		// Create new bucket
		b = &bucket{
			tokens:     rl.limit - 1,
			lastRefill: time.Now(),
		}
		rl.mu.Lock()
		rl.requests[key] = b
		rl.mu.Unlock()
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	// Refill tokens if window has passed
	if elapsed >= rl.window {
		b.tokens = rl.limit
		b.lastRefill = now
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, first is client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is "IP:port", so strip the port
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// RateLimit creates a middleware that limits requests per IP address
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use client IP as the key
			key := getClientIP(r)

			if !limiter.allow(key) {
				// Set rate limit headers to inform the client of the limits
				w.Header().Set("X-RateLimit-Limit", fmt.Sprint(limit))
				w.Header().Set("X-RateLimit-Window", window.String())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":       "rate limit exceeded",
					"retry_after": window.String(),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByUser creates a middleware that limits requests per authenticated user.
// This should be used after the JWTAuth middleware. If the user is not authenticated,
// it falls back to IP-based rate limiting.
func RateLimitByUser(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string

			// Get user ID from context (set by JWTAuth middleware)
			userID, err := auth.GetUserIDOrError(r.Context())
			if err != nil {
				// If user is not authenticated, fall back to IP-based rate limiting
				key = getClientIP(r)
			} else {
				// Use user ID as the key for authenticated users
				key = userID.String()
			}

			if !limiter.allow(key) {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprint(limit))
				w.Header().Set("X-RateLimit-Window", window.String())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":       "rate limit exceeded",
					"retry_after": window.String(),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// KeyFunc is a function that extracts a rate limit key from a request
type KeyFunc func(*http.Request) string

// RateLimitWithKey creates a middleware that limits requests using a custom key function.
// This allows for flexible rate limiting strategies based on any request attribute.
func RateLimitWithKey(limit int, window time.Duration, keyFunc KeyFunc) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			if !limiter.allow(key) {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprint(limit))
				w.Header().Set("X-RateLimit-Window", window.String())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":       "rate limit exceeded",
					"retry_after": window.String(),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitAuth creates a rate limiter specifically for authentication endpoints
// This is more restrictive to prevent brute force attacks
func RateLimitAuth(limit int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimit(limit, window)
}
