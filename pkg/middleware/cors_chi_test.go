package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestCORSChiTopLevel verifies that applying CORS as a top-level chi middleware
// (the pattern used by every browser-facing Cairn service router) reliably:
//   - answers browser preflight OPTIONS for an Authorization request, even though
//     the route only registers POST (chi would otherwise return 405); and
//   - emits Access-Control-Allow-Origin on a plain GET health endpoint.
//
// This guards the wiring assumption behind task_1318: top-level Use runs before
// chi's route/method resolution.
func TestCORSChiTopLevel(t *testing.T) {
	r := chi.NewRouter()
	r.Use(CORS(DefaultCORSConfig()))

	// Health endpoint (browser does a simple GET, no preflight).
	r.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// An auth-style endpoint registered only for POST — OPTIONS is never registered.
	r.Post("/api/v1/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("preflight OPTIONS for Authorization request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
		req.Header.Set("Origin", "https://web.example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d, want %d", w.Code, http.StatusNoContent)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got == "" {
			t.Error("preflight missing Access-Control-Allow-Origin")
		}
		allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
		if !strings.Contains(strings.ToLower(allowHeaders), "authorization") {
			t.Errorf("Access-Control-Allow-Headers = %q, want it to contain 'authorization'", allowHeaders)
		}
	})

	t.Run("GET health carries CORS origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		req.Header.Set("Origin", "https://web.example.com")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("health status = %d, want %d", w.Code, http.StatusOK)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
		}
	})
}
