package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgauth "github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/google/uuid"
)

// mockVoteRepository is a test double for VoteRepositoryInterface.
// Only the methods exercised by the handler under test need non-nil implementations.
type mockVoteRepository struct {
	db.VoteRepositoryInterface // embed to satisfy remaining interface methods

	getUserVoteStatsFunc func(ctx context.Context, userID string) (int, int, error)
}

func (m *mockVoteRepository) GetUserVoteStats(ctx context.Context, userID string) (int, int, error) {
	return m.getUserVoteStatsFunc(ctx, userID)
}

// newTestServer creates a minimal Server suitable for handler unit tests.
// Fields not required for the test are left nil.
func newTestServer(voteRepo db.VoteRepositoryInterface) *Server {
	return &Server{
		voteRepo: voteRepo,
	}
}

// withUserID returns a copy of ctx carrying the given UUID as the authenticated user.
func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return pkgauth.SetUserIDInContext(ctx, id)
}

// TestHandleGetUserVoteStats_Success checks the happy path: valid auth + successful DB call.
func TestHandleGetUserVoteStats_Success(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := &mockVoteRepository{
		getUserVoteStatsFunc: func(_ context.Context, gotUserID string) (int, int, error) {
			if gotUserID != userID.String() {
				t.Errorf("expected userID %s, got %s", userID.String(), gotUserID)
			}
			return 10, 3, nil
		},
	}

	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/user/vote-stats", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	w := httptest.NewRecorder()

	srv.handleGetUserVoteStats(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Data struct {
			Upvotes   int `json:"upvotes"`
			Downvotes int `json:"downvotes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Data.Upvotes != 10 {
		t.Errorf("expected upvotes 10, got %d", body.Data.Upvotes)
	}
	if body.Data.Downvotes != 3 {
		t.Errorf("expected downvotes 3, got %d", body.Data.Downvotes)
	}
}

// TestHandleGetUserVoteStats_NoAuth checks that a missing user ID in context returns 500
// (the auth middleware normally prevents this; this guards against programming errors).
func TestHandleGetUserVoteStats_NoAuth(t *testing.T) {
	srv := newTestServer(&mockVoteRepository{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/user/vote-stats", nil)
	// No user ID injected into context
	w := httptest.NewRecorder()

	srv.handleGetUserVoteStats(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when user ID missing from context, got %d", w.Result().StatusCode)
	}
}

// TestHandleGetUserVoteStats_DBError checks that a repository error propagates as 500.
func TestHandleGetUserVoteStats_DBError(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	repo := &mockVoteRepository{
		getUserVoteStatsFunc: func(_ context.Context, _ string) (int, int, error) {
			return 0, 0, context.DeadlineExceeded
		},
	}

	srv := newTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/user/vote-stats", nil)
	req = req.WithContext(withUserID(req.Context(), userID))
	w := httptest.NewRecorder()

	srv.handleGetUserVoteStats(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error, got %d", w.Result().StatusCode)
	}
}
