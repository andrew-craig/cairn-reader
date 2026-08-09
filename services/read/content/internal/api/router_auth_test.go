package api

import (
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// newTestRouterHandler builds a real router with a lazily-opened (never-dialed)
// *sql.DB. The routes under test reject unauthenticated/unauthorized requests in
// the auth middleware, before any query would run, so no live Postgres is needed.
func newTestRouterHandler(t *testing.T) (http.Handler, *auth.Middleware, *auth.InternalAuthMiddleware) {
	t.Helper()

	sqlDB, err := sql.Open("postgres", "postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	validator := auth.NewValidator(&privateKey.PublicKey)
	authMiddleware := auth.NewMiddleware(validator)
	internalAuthMiddleware := auth.NewInternalAuthMiddleware("test-internal-key")

	router := NewRouter(&database.DB{DB: sqlDB}, "", "", "test-internal-key", authMiddleware, internalAuthMiddleware)
	return router, authMiddleware, internalAuthMiddleware
}

// httpsRequest builds a request satisfying RequireHTTPS (X-Forwarded-Proto: https)
// for the given method/path, with an optional body.
func httpsRequest(method, path string, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	return req
}

// TestUnauthenticatedRoutesRejected proves C1/C2 are closed: every route that used
// to accept anonymous requests now requires credentials.
//
// C1 (SSRF via /detect, /discover-feed): these are the web/mobile "add link"
// flows and are only reachable once a user is logged in, so they require a JWT.
// C2 (anonymous writes to shared content records via /, /{id}, /bulk,
// /check-duplicate): these are internal service-to-service routes (Ingest RSS,
// Email Ingest), so they require the internal API key.
func TestUnauthenticatedRoutesRejected(t *testing.T) {
	router, _, _ := newTestRouterHandler(t)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"detect", http.MethodPost, "/api/v1/content/detect", `{"url":"https://example.com"}`},
		{"discover-feed", http.MethodPost, "/api/v1/content/discover-feed", `{"url":"https://example.com"}`},
		{"create", http.MethodPost, "/api/v1/content", `{"source_url":"https://example.com"}`},
		{"get by id", http.MethodGet, "/api/v1/content/11111111-1111-1111-1111-111111111111", ""},
		{"update", http.MethodPut, "/api/v1/content/11111111-1111-1111-1111-111111111111", `{"url":"https://example.com","html":"<p>x</p>"}`},
		{"bulk create", http.MethodPost, "/api/v1/content/bulk", `{"contents":[]}`},
		{"check duplicate", http.MethodPost, "/api/v1/content/check-duplicate", `{"items":[]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httpsRequest(tc.method, tc.path, tc.body)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s: expected 401 with no credentials, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}

// routeAllowlist enumerates the only routes on this router that intentionally
// require no per-request authentication. Any other route MUST carry either
// RequireAuth or RequireInternalAPIKey in its middleware chain — this is the
// ratchet that fails a future PR that registers a new unauthenticated route.
var routeAllowlist = map[string]bool{
	"GET /health/live":  true,
	"GET /health/ready": true,
	"GET /":             true,
}

var routeParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// TestRouterInventory_UnlistedRoutesRequireAuth walks every registered route and
// asserts it is either explicitly allowlisted as public or has RequireAuth /
// RequireInternalAPIKey in its middleware chain.
func TestRouterInventory_UnlistedRoutesRequireAuth(t *testing.T) {
	router, authMiddleware, internalAuthMiddleware := newTestRouterHandler(t)

	mux, ok := router.(chi.Router)
	require.True(t, ok, "NewRouter must return a chi.Router for route introspection")

	requireAuthPtr := reflect.ValueOf(authMiddleware.RequireAuth).Pointer()
	requireInternalPtr := reflect.ValueOf(internalAuthMiddleware.RequireInternalAPIKey).Pointer()

	err := chi.Walk(mux, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		key := method + " " + routeParamPattern.ReplaceAllString(route, "{param}")
		if routeAllowlist[key] {
			return nil
		}

		for _, mw := range middlewares {
			p := reflect.ValueOf(mw).Pointer()
			if p == requireAuthPtr || p == requireInternalPtr {
				return nil
			}
		}

		t.Errorf("route %s is not in the public allowlist and has no auth middleware applied", key)
		return nil
	})
	require.NoError(t, err)
}
