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

func TestCORSForOrigins(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := CORSForOrigins("http://allowed.com", "http://other.com")
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://allowed.com")
	w := httptest.NewRecorder()

	mw(handler).ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
		t.Error("expected specific origin for CORSForOrigins")
	}
}

func TestDevelopmentCORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("sets wildcard for any origin", func(t *testing.T) {
		mw := DevelopmentCORS()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://anything.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected wildcard origin")
		}
	})

	t.Run("handles preflight", func(t *testing.T) {
		mw := DevelopmentCORS()
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204 for preflight, got %d", w.Code)
		}
	})
}

func TestProductionCORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := ProductionCORS([]string{"http://prod.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://prod.example.com")
	w := httptest.NewRecorder()

	mw(handler).ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://prod.example.com" {
		t.Error("expected production origin")
	}
}

func TestCORSWithWildcard(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("matches wildcard subdomain", func(t *testing.T) {
		mw := CORSWithWildcard([]string{"*.example.com"})
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://app.example.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "http://app.example.com" {
			t.Error("expected wildcard subdomain match")
		}
	})

	t.Run("blocks non-matching origin", func(t *testing.T) {
		mw := CORSWithWildcard([]string{"*.example.com"})
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://evil.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set origin for non-matching domain")
		}
	})

	t.Run("blocks suffix-bypass origin sharing no dot boundary", func(t *testing.T) {
		mw := CORSWithWildcard([]string{"*.example.com"})
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://evilexample.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set origin for a host that merely shares the domain as a suffix without a dot boundary")
		}
	})

	t.Run("blocks lookalike origin with domain as a prefix of a longer host", func(t *testing.T) {
		mw := CORSWithWildcard([]string{"*.example.com"})
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com.evil.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set origin for a host where the allowed domain appears as a prefix segment")
		}
	})

	t.Run("handles preflight for wildcard", func(t *testing.T) {
		mw := CORSWithWildcard([]string{"*.example.com"})
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://app.example.com")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("expected Allow-Methods on preflight")
		}
	})
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		allowed  []string
		expected bool
	}{
		{"exact match", "http://example.com", []string{"http://example.com"}, true},
		{"wildcard", "http://anything.com", []string{"*"}, true},
		{"wildcard subdomain", "http://app.example.com", []string{"*.example.com"}, true},
		{"no match", "http://evil.com", []string{"http://example.com"}, false},
		{"suffix-bypass without dot boundary", "http://evilexample.com", []string{"*.example.com"}, false},
		{"domain as prefix segment of longer host", "http://example.com.evil.com", []string{"*.example.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOriginAllowed(tt.origin, tt.allowed); got != tt.expected {
				t.Errorf("isOriginAllowed(%q, %v) = %v, want %v", tt.origin, tt.allowed, got, tt.expected)
			}
		})
	}
}
