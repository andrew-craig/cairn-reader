package api

import (
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// newTestServerHandler builds a real router via Server.Routes() with nil
// repositories/db. The routes under test reject unauthenticated requests in
// the auth middleware, before any repository or database call would happen,
// so no live Postgres is needed.
func newTestServerHandler(t *testing.T) (http.Handler, *auth.Middleware, *auth.InternalAuthMiddleware) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	validator := auth.NewValidator(&privateKey.PublicKey)
	authMiddleware := auth.NewMiddleware(validator)
	internalAuthMiddleware := auth.NewInternalAuthMiddleware("test-internal-key")

	server := NewServer(nil, nil, nil, nil, nil, authMiddleware, internalAuthMiddleware, slog.Default())
	return server.Routes(), authMiddleware, internalAuthMiddleware
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

// TestArticleSubmissionRequiresInternalAPIKey proves the explore auth finding
// is closed: POST /api/v1/explore/article, previously reachable by anyone,
// now requires the internal service-to-service API key.
func TestArticleSubmissionRequiresInternalAPIKey(t *testing.T) {
	router, _, _ := newTestServerHandler(t)

	req := httpsRequest(http.MethodPost, "/api/v1/explore/article", `{"articles":[]}`)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/v1/explore/article: expected 401 with no credentials, got %d: %s", w.Code, w.Body.String())
	}
}

// routeAllowlist enumerates the only routes on this router that intentionally
// require no per-request authentication. Any other route MUST carry either
// RequireAuth or RequireInternalAPIKey in its middleware chain — this is the
// ratchet that fails a future PR that registers a new unauthenticated route.
var routeAllowlist = map[string]bool{
	"GET /health/live":   true,
	"HEAD /health/live":  true,
	"GET /health/ready":  true,
	"HEAD /health/ready": true,
}

var routeParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// TestRouterInventory_UnlistedRoutesRequireAuth walks every registered route and
// asserts it is either explicitly allowlisted as public or has RequireAuth /
// RequireInternalAPIKey in its middleware chain.
func TestRouterInventory_UnlistedRoutesRequireAuth(t *testing.T) {
	router, authMiddleware, internalAuthMiddleware := newTestServerHandler(t)

	mux, ok := router.(chi.Router)
	require.True(t, ok, "Routes() must return a chi.Router for route introspection")

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
