package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/recommend"
	"github.com/google/uuid"
)

// mockArticleRepo is a minimal mock implementing ArticleRepositoryInterface for handler tests.
type mockArticleRepo struct {
	db.ArticleRepositoryInterface
	searchResults []models.Article
	searchErr     error
}

func (m *mockArticleRepo) Search(_ context.Context, _ string, _ int, _ int) ([]models.Article, error) {
	return m.searchResults, m.searchErr
}

func newTestServer(articleRepo db.ArticleRepositoryInterface) *Server {
	return &Server{
		articleRepo: articleRepo,
		// voteRepo, userRepo, engine, authMiddleware, logger left nil;
		// handleSearch only touches articleRepo.
	}
}

// requestWithUser injects a fixed userID into context so auth middleware is bypassed in tests.
func requestWithUser(r *http.Request) *http.Request {
	ctx := auth.SetUserIDInContext(r.Context(), uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	return r.WithContext(ctx)
}

func TestHandleSearch_MissingQ(t *testing.T) {
	s := newTestServer(&mockArticleRepo{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/search", nil)
	req = requestWithUser(req)
	w := httptest.NewRecorder()

	s.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "validation_error" {
		t.Errorf("expected validation_error, got %v", body["error"])
	}
}

func TestHandleSearch_EmptyResults(t *testing.T) {
	s := newTestServer(&mockArticleRepo{searchResults: []models.Article{}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/search?q=golang", nil)
	req = requestWithUser(req)
	w := httptest.NewRecorder()

	s.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %T", body["data"])
	}
	count, ok := data["count"].(float64)
	if !ok || count != 0 {
		t.Errorf("expected count 0, got %v", data["count"])
	}
}

func TestHandleSearch_PaginationDefaults(t *testing.T) {
	repo := &mockArticleRepo{searchResults: []models.Article{}}
	s := newTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/search?q=test", nil)
	req = requestWithUser(req)
	w := httptest.NewRecorder()

	s.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data := body["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	if pagination["limit"].(float64) != 20 {
		t.Errorf("expected default limit 20, got %v", pagination["limit"])
	}
	if pagination["offset"].(float64) != 0 {
		t.Errorf("expected default offset 0, got %v", pagination["offset"])
	}
}

func TestHandleSearch_CustomPagination(t *testing.T) {
	repo := &mockArticleRepo{searchResults: []models.Article{}}
	s := newTestServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/explore/search?q=test&limit=5&offset=10", nil)
	req = requestWithUser(req)
	w := httptest.NewRecorder()

	s.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data := body["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	if pagination["limit"].(float64) != 5 {
		t.Errorf("expected limit 5, got %v", pagination["limit"])
	}
	if pagination["offset"].(float64) != 10 {
		t.Errorf("expected offset 10, got %v", pagination["offset"])
	}
}

// Satisfy the compiler: unused fields reference
var _ *recommend.Engine = nil
