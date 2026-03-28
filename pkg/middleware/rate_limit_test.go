package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		rl := NewRateLimiter(3, 1*time.Second)

		if !rl.allow("key") {
			t.Error("first request should be allowed")
		}
		if !rl.allow("key") {
			t.Error("second request should be allowed")
		}
		if !rl.allow("key") {
			t.Error("third request should be allowed")
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		rl := NewRateLimiter(2, 1*time.Second)

		rl.allow("key")
		rl.allow("key")

		if rl.allow("key") {
			t.Error("third request should be blocked")
		}
	})

	t.Run("refills after window", func(t *testing.T) {
		rl := NewRateLimiter(1, 50*time.Millisecond)

		rl.allow("key")
		if rl.allow("key") {
			t.Error("should be blocked before window")
		}

		time.Sleep(60 * time.Millisecond)

		if !rl.allow("key") {
			t.Error("should be allowed after window")
		}
	})

	t.Run("tracks keys independently", func(t *testing.T) {
		rl := NewRateLimiter(1, 1*time.Second)

		rl.allow("key1")
		if rl.allow("key1") {
			t.Error("key1 should be blocked")
		}

		if !rl.allow("key2") {
			t.Error("key2 should be allowed")
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		rl := NewRateLimiter(100, 1*time.Second)
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rl.allow("concurrent")
			}()
		}

		wg.Wait()
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		mw := RateLimit(2, 1*time.Second)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			mw(handler).ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Errorf("request %d should not be rate limited", i+1)
			}
		}
	})

	t.Run("returns 429 when limit exceeded", func(t *testing.T) {
		mw := RateLimit(1, 1*time.Second)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// First request passes
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w1 := httptest.NewRecorder()
		mw(handler).ServeHTTP(w1, req1)

		// Second request blocked
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		mw(handler).ServeHTTP(w2, req2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", w2.Code)
		}

		if w2.Header().Get("X-RateLimit-Window") == "" {
			t.Error("expected rate limit headers")
		}
	})
}
