package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/database"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// newTestRouterHandler builds a real router with a lazily-opened (never-dialed)
// *sql.DB. Every route under test is rejected by the internal-auth middleware
// before any query would run, so no live Postgres is needed.
func newTestRouterHandler(t *testing.T) (http.Handler, *auth.InternalAuthMiddleware) {
	t.Helper()

	sqlDB, err := sql.Open("postgres", "postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	internalAuthMiddleware := auth.NewInternalAuthMiddleware("test-internal-key")

	router := NewRouter(&database.DB{DB: sqlDB}, internalAuthMiddleware)
	return router, internalAuthMiddleware
}

// httpsRequest builds a request satisfying RequireHTTPS (X-Forwarded-Proto: https)
// for the given method/path.
func httpsRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	return req
}

// TestUnauthenticatedRoutesRejected proves the P2-IDOR finding is closed: every
// user-scoped route that used to trust user_id from the path with zero inbound
// auth now requires the internal API key, including a cross-tenant request that
// simply names a different user_id in the URL.
func TestUnauthenticatedRoutesRejected(t *testing.T) {
	router, _ := newTestRouterHandler(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"subscribe", http.MethodPost, "/api/v1/source/rss/user/11111111-1111-1111-1111-111111111111/subscription"},
		{"list subscriptions", http.MethodGet, "/api/v1/source/rss/user/11111111-1111-1111-1111-111111111111/subscription"},
		{"unsubscribe", http.MethodDelete, "/api/v1/source/rss/user/11111111-1111-1111-1111-111111111111/subscription/22222222-2222-2222-2222-222222222222"},
		{"cross-tenant list subscriptions", http.MethodGet, "/api/v1/source/rss/user/99999999-9999-9999-9999-999999999999/subscription"},
		{"update feed", http.MethodPatch, "/api/v1/source/rss/feed/22222222-2222-2222-2222-222222222222"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httpsRequest(tc.method, tc.path)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s: expected 401 with no credentials, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}

// routeAllowlist enumerates the only routes on this router that intentionally
// require no per-request authentication. Any other route MUST carry
// RequireInternalAPIKey in its middleware chain — this is the ratchet that
// fails a future PR that registers a new unauthenticated route.
var routeAllowlist = map[string]bool{
	"GET /health/live":  true,
	"GET /health/ready": true,
	"GET /":             true,
}

var routeParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// TestRouterInventory_UnlistedRoutesRequireAuth walks every registered route and
// asserts it is either explicitly allowlisted as public or has
// RequireInternalAPIKey in its middleware chain.
func TestRouterInventory_UnlistedRoutesRequireAuth(t *testing.T) {
	router, internalAuthMiddleware := newTestRouterHandler(t)

	mux, ok := router.(chi.Router)
	require.True(t, ok, "NewRouter must return a chi.Router for route introspection")

	requireInternalPtr := reflect.ValueOf(internalAuthMiddleware.RequireInternalAPIKey).Pointer()

	err := chi.Walk(mux, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		key := method + " " + routeParamPattern.ReplaceAllString(route, "{param}")
		if routeAllowlist[key] {
			return nil
		}

		for _, mw := range middlewares {
			p := reflect.ValueOf(mw).Pointer()
			if p == requireInternalPtr {
				return nil
			}
		}

		t.Errorf("route %s is not in the public allowlist and has no auth middleware applied", key)
		return nil
	})
	require.NoError(t, err)
}
