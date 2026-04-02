package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireHTTPS(t *testing.T) {
	handler := RequireHTTPS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("allows X-Forwarded-Proto https", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("rejects plain HTTP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})
}

func TestRequireHTTPSWithRedirect(t *testing.T) {
	handler := RequireHTTPSWithRedirect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("allows X-Forwarded-Proto https", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("redirects plain HTTP to HTTPS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Host = "example.com"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMovedPermanently {
			t.Errorf("expected 301, got %d", w.Code)
		}
		location := w.Header().Get("Location")
		if location != "https://example.com/test" {
			t.Errorf("expected redirect to https, got %s", location)
		}
	})
}

func TestSecureHeaders(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":   "default-src 'self'",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}

	for header, expected := range headers {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, got)
		}
	}
}

func TestSecureHeadersRelaxed(t *testing.T) {
	handler := SecureHeadersRelaxed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should have these headers
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}

	// Should NOT have CSP (relaxed version)
	if w.Header().Get("Content-Security-Policy") != "" {
		t.Error("relaxed version should not set CSP")
	}
}

func TestValidateContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("skips GET requests", func(t *testing.T) {
		mw := ValidateContentType("application/json")
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("rejects wrong content type on POST", func(t *testing.T) {
		mw := ValidateContentType("application/json")
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected 415, got %d", w.Code)
		}
	})

	t.Run("accepts correct content type with charset", func(t *testing.T) {
		mw := ValidateContentType("application/json")
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		w := httptest.NewRecorder()

		mw(handler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

func TestPreventCaching(t *testing.T) {
	handler := PreventCaching(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Cache-Control") == "" {
		t.Error("expected Cache-Control header")
	}
	if w.Header().Get("Pragma") != "no-cache" {
		t.Error("expected Pragma: no-cache")
	}
}

func TestAllowCaching(t *testing.T) {
	handler := AllowCaching(3600)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Errorf("expected public caching, got %s", w.Header().Get("Cache-Control"))
	}
}
