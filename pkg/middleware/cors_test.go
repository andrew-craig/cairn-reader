package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("sets wildcard origin", func(t *testing.T) {
		mw := CORS(DefaultCORSConfig())
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected wildcard origin")
		}
	})

	t.Run("handles preflight OPTIONS", func(t *testing.T) {
		mw := CORS(DefaultCORSConfig())
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("strict config sets specific origin", func(t *testing.T) {
		config := StrictCORSConfig([]string{"http://allowed.com"})
		mw := CORS(config)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://allowed.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
			t.Error("expected specific origin")
		}
		if w.Header().Get("Vary") != "Origin" {
			t.Error("expected Vary: Origin")
		}
	})

	t.Run("strict config blocks disallowed origin", func(t *testing.T) {
		config := StrictCORSConfig([]string{"http://allowed.com"})
		mw := CORS(config)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://notallowed.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set Access-Control-Allow-Origin for disallowed origin")
		}
	})

	t.Run("no origin header skips CORS", func(t *testing.T) {
		mw := CORS(DefaultCORSConfig())
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set CORS headers without Origin")
		}
	})
}
