package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/sync"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// stubPinger always reports healthy, so readiness tests don't need a real DB.
type stubPinger struct{}

func (stubPinger) Ping(ctx context.Context) error { return nil }

// stubFeedRepo is a minimal FeedRepositoryInterface stub. The routes under
// test reject unauthenticated requests in the auth middleware before any
// repository call would happen, so the methods are never exercised.
type stubFeedRepo struct{}

func (stubFeedRepo) GetNextFeed(ctx context.Context) (*models.Feed, error) { return nil, nil }
func (stubFeedRepo) UpdateFetchResult(ctx context.Context, feedID int, success bool, etag, lastModified string) error {
	return nil
}
func (stubFeedRepo) ImportFeeds(ctx context.Context, feedURLs []string) error { return nil }
func (stubFeedRepo) ListFeeds(ctx context.Context, enabledOnly bool) ([]models.Feed, error) {
	return nil, nil
}
func (stubFeedRepo) RecordFetchHistory(ctx context.Context, feedID int, success bool, articlesFound, articlesSent int, errorMsg string) error {
	return nil
}
func (stubFeedRepo) GetFeedByID(ctx context.Context, feedID int) (*models.Feed, error) {
	return nil, nil
}
func (stubFeedRepo) GetFeedStats(ctx context.Context) (total, enabled, disabled, neverFetched int, err error) {
	return 0, 0, 0, 0, nil
}

// stubRecommenderClient is a no-op RecommenderClientInterface stub.
type stubRecommenderClient struct{}

func (stubRecommenderClient) SubmitArticles(ctx context.Context, articles []models.Article) error {
	return nil
}

// newTestRouterHandler builds a real router via NewRouter with stub
// dependencies. The routes under test reject unauthenticated requests in the
// internal-auth middleware before any dependency is touched.
func newTestRouterHandler(t *testing.T) (http.Handler, *auth.InternalAuthMiddleware) {
	t.Helper()

	repo := stubFeedRepo{}
	feedFetcher := fetcher.NewFetcher(repo, stubRecommenderClient{}, time.Minute)
	feedSyncer := sync.NewFeedSyncer(repo, "", "https://example.com/feeds.txt")
	internalAuthMiddleware := auth.NewInternalAuthMiddleware("test-internal-key")

	router := NewRouter(context.Background(), stubPinger{}, repo, feedFetcher, feedSyncer, internalAuthMiddleware, slog.Default())
	return router, internalAuthMiddleware
}

// httpsRequest builds a request satisfying RequireHTTPS (X-Forwarded-Proto: https).
func httpsRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	return req
}

// TestTriggerRoutesRequireInternalAPIKey proves the explore auth finding is
// closed: the fetcher's manual trigger endpoints, previously reachable by
// anyone, now require the internal service-to-service API key.
func TestTriggerRoutesRequireInternalAPIKey(t *testing.T) {
	router, _ := newTestRouterHandler(t)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"fetch", http.MethodPost, "/api/v1/explore/feed/fetch"},
		{"stats", http.MethodGet, "/api/v1/explore/feed/stats"},
		{"sync", http.MethodPost, "/api/v1/explore/feed/sync"},
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
	"GET /health/live":   true,
	"HEAD /health/live":  true,
	"GET /health/ready":  true,
	"HEAD /health/ready": true,
}

var routeParamPattern = regexp.MustCompile(`\{[^}]+\}`)

// TestRouterInventory_UnlistedRoutesRequireAuth walks every registered route
// and asserts it is either explicitly allowlisted as public or has
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
