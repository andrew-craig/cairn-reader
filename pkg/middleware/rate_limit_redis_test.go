package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockRedisScripter implements RedisScripter for testing.
type mockRedisScripter struct {
	result int64
	err    error
	// lastKey records the last key passed for inspection
	lastKey string
}

func (m *mockRedisScripter) EvalInt64(_ context.Context, _ string, keys []string, _ ...interface{}) (int64, error) {
	if len(keys) > 0 {
		m.lastKey = keys[0]
	}
	return m.result, m.err
}

func TestRedisRateLimiter_Allow(t *testing.T) {
	t.Run("allows when redis returns 1", func(t *testing.T) {
		client := &mockRedisScripter{result: 1}
		rl := NewRedisRateLimiter(client, 5, time.Minute)

		if !rl.allow("key1") {
			t.Error("expected request to be allowed")
		}
	})

	t.Run("denies when redis returns 0", func(t *testing.T) {
		client := &mockRedisScripter{result: 0}
		rl := NewRedisRateLimiter(client, 5, time.Minute)

		if rl.allow("key1") {
			t.Error("expected request to be denied")
		}
	})

	t.Run("fails open on redis error", func(t *testing.T) {
		client := &mockRedisScripter{err: errors.New("redis unavailable")}
		rl := NewRedisRateLimiter(client, 5, time.Minute)

		// Should allow the request (fail open) to avoid blocking all traffic
		if !rl.allow("key1") {
			t.Error("expected fail-open on redis error")
		}
	})

	t.Run("uses prefixed key", func(t *testing.T) {
		client := &mockRedisScripter{result: 1}
		rl := NewRedisRateLimiter(client, 5, time.Minute)

		rl.allow("192.168.1.1")

		if client.lastKey != "ratelimit:192.168.1.1" {
			t.Errorf("expected prefixed key, got %q", client.lastKey)
		}
	})
}

func TestRateLimitRedisMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allows request when redis permits", func(t *testing.T) {
		client := &mockRedisScripter{result: 1}
		mw := RateLimitRedis(client, 5, time.Minute)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("returns 429 when redis denies", func(t *testing.T) {
		client := &mockRedisScripter{result: 0}
		mw := RateLimitRedis(client, 5, time.Minute)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", w.Code)
		}
		if w.Header().Get("X-RateLimit-Limit") == "" {
			t.Error("expected X-RateLimit-Limit header")
		}
		if w.Header().Get("X-RateLimit-Window") == "" {
			t.Error("expected X-RateLimit-Window header")
		}
	})

	t.Run("fails open on redis error", func(t *testing.T) {
		client := &mockRedisScripter{err: errors.New("redis unavailable")}
		mw := RateLimitRedis(client, 5, time.Minute)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		w := httptest.NewRecorder()
		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 (fail open), got %d", w.Code)
		}
	})
}
