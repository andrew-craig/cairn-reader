package db

import (
	"context"
	"testing"
	"time"
)

// TestSearch_EmptyQuery exercises the Search path when a DB is available.
// It skips silently when no test Postgres is reachable (same pattern as the
// other tests in this package).
func TestSearch_ReturnsResults(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	now := time.Now()
	articles := []struct {
		id    string
		link  string
		title string
	}{
		{"s-id-1", "https://example.com/search1", "Go programming tutorial"},
		{"s-id-2", "https://example.com/search2", "Python basics"},
		{"s-id-3", "https://example.com/search3", "Advanced Go patterns"},
	}

	for _, a := range articles {
		art := createTestArticle(a.id, a.link, a.title)
		art.Published = now
		if err := repo.Create(ctx, art); err != nil {
			t.Fatalf("failed to create article %s: %v", a.id, err)
		}
	}

	results, err := repo.Search(ctx, "go", 10, 0)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	// "Go programming tutorial" and "Advanced Go patterns" should match
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'go', got %d", len(results))
	}
}

func TestSearch_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 5; i++ {
		art := createTestArticle(
			"p-id-"+string(rune('1'+i)),
			"https://example.com/page"+string(rune('1'+i)),
			"Rust article",
		)
		art.Published = now.Add(time.Duration(i) * time.Minute)
		if err := repo.Create(ctx, art); err != nil {
			t.Fatalf("failed to create article: %v", err)
		}
	}

	page1, err := repo.Search(ctx, "rust", 2, 0)
	if err != nil {
		t.Fatalf("Search page 1 error: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 results on page 1, got %d", len(page1))
	}

	page2, err := repo.Search(ctx, "rust", 2, 2)
	if err != nil {
		t.Fatalf("Search page 2 error: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 results on page 2, got %d", len(page2))
	}

	// IDs should not overlap
	ids1 := map[string]bool{}
	for _, a := range page1 {
		ids1[a.ID] = true
	}
	for _, a := range page2 {
		if ids1[a.ID] {
			t.Errorf("article %s appeared on both pages", a.ID)
		}
	}
}

func TestSearch_DeletedArticlesExcluded(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	art := createTestArticle("del-1", "https://example.com/del1", "Deleted typescript post")
	if err := repo.Create(ctx, art); err != nil {
		t.Fatalf("failed to create article: %v", err)
	}
	if _, err := db.Exec(ctx, "UPDATE articles SET deleted = true WHERE id = $1", art.ID); err != nil {
		t.Fatalf("failed to soft-delete article: %v", err)
	}

	results, err := repo.Search(ctx, "typescript", 10, 0)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (deleted article), got %d", len(results))
	}
}
