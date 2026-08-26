package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter.
type RateLimiter struct {
	requests map[string]*bucket
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	cleanup  time.Duration
}

// bucket represents a token bucket for a specific client.
type bucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter.
// limit: maximum number of requests allowed.
// window: time window for the limit (e.g., 1 minute).
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*bucket),
		limit:    limit,
		window:   window,
		cleanup:  window * 2,
	}

	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes old entries to prevent memory leaks.
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

// allow checks if a request should be allowed.
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.RLock()
	b, exists := rl.requests[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		b, exists = rl.requests[key]
		if !exists {
			b = &bucket{
				tokens:     rl.limit,
				lastRefill: time.Now(),
			}
			rl.requests[key] = b
		}
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)

	if elapsed >= rl.window {
		b.tokens = rl.limit
		b.lastRefill = now
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// RateLimit creates a middleware that limits requests per IP address.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := getClientIP(r)

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

// RateLimitRedis creates a middleware that limits requests per IP address using
// a Redis-backed sliding window algorithm. Safe for multi-instance deployments.
// Falls back to allowing the request if Redis is unavailable.
func RateLimitRedis(client RedisScripter, limit int, window time.Duration) func(http.Handler) http.Handler {
	limiter := NewRedisRateLimiter(client, limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := getClientIP(r)

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

// KeyFunc is a function that extracts a rate limit key from a request.
type KeyFunc func(*http.Request) string

// RateLimitWithKey creates a middleware that limits requests using a custom key function.
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
