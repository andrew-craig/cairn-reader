package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/andrew-craig/cairn/pkg/models"
	_ "github.com/lib/pq"
)

// setupTestDB creates a test database connection and runs migrations
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Connect to test database
	// In a real scenario, you'd use a test database or docker container
	// For this test, we'll use a local PostgreSQL instance
	connStr := "host=localhost port=5432 user=cairn password=cairn_password dbname=cairn_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("error closing database: %v", closeErr)
		}
		t.Skipf("Skipping test: could not ping test database: %v", err)
	}

	// Clean up test data
	cleanupTestDB(t, db)

	return db
}

// cleanupTestDB removes all test data from the database
func cleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	queries := []string{
		"DELETE FROM user_articles",
		"DELETE FROM articles",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Logf("Warning: failed to clean up: %v", err)
		}
	}
}

// teardownTestDB cleans up and closes the database connection
func teardownTestDB(t *testing.T, db *sql.DB) {
	t.Helper()
	cleanupTestDB(t, db)
	if err := db.Close(); err != nil {
		t.Logf("error closing database: %v", err)
	}
}

// createTestArticle creates a test article with default values
func createTestArticle(id, link, title string) models.Article {
	return models.Article{
		ID:          id,
		Title:       title,
		Link:        link,
		Description: "Test description",
		Content:     "Test content",
		Author:      "Test Author",
		Published:   time.Now(),
		FeedURL:     "https://example.com/feed",
		FeedTitle:   "Test Feed",
		Categories:  []string{"tech", "programming"},
	}
}

// TestCreate_NewArticle tests inserting a new article
func TestCreate_NewArticle(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	article := createTestArticle("test-id-1", "https://example.com/article1", "Test Article 1")

	err := repo.Create(ctx, article)
	if err != nil {
		t.Fatalf("failed to create article: %v", err)
	}

	// Verify the article was created
	retrieved, err := repo.GetByID(ctx, article.ID)
	if err != nil {
		t.Fatalf("failed to retrieve article: %v", err)
	}

	if retrieved.ID != article.ID {
		t.Errorf("expected ID %s, got %s", article.ID, retrieved.ID)
	}
	if retrieved.Link != article.Link {
		t.Errorf("expected link %s, got %s", article.Link, retrieved.Link)
	}
	if retrieved.Title != article.Title {
		t.Errorf("expected title %s, got %s", article.Title, retrieved.Title)
	}
	if retrieved.Upvotes != 0 {
		t.Errorf("expected upvotes 0, got %d", retrieved.Upvotes)
	}
	if retrieved.Downvotes != 0 {
		t.Errorf("expected downvotes 0, got %d", retrieved.Downvotes)
	}
	if retrieved.Recommends != 0 {
		t.Errorf("expected recommends 0, got %d", retrieved.Recommends)
	}
	if retrieved.Deleted {
		t.Errorf("expected deleted false, got true")
	}
}

// TestCreate_DuplicateLink_UpdatesArticle tests deduplication based on link
func TestCreate_DuplicateLink_UpdatesArticle(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	// Create initial article
	article1 := createTestArticle("test-id-1", "https://example.com/article1", "Original Title")
	err := repo.Create(ctx, article1)
	if err != nil {
		t.Fatalf("failed to create initial article: %v", err)
	}

	// Manually set some engagement metrics
	_, err = db.Exec("UPDATE articles SET upvotes = 5, downvotes = 2, recommends = 10 WHERE id = $1", article1.ID)
	if err != nil {
		t.Fatalf("failed to update metrics: %v", err)
	}

	// Create duplicate article with same link but different ID and title
	article2 := createTestArticle("test-id-2", "https://example.com/article1", "Updated Title")
	article2.Description = "Updated description"
	article2.Content = "Updated content"

	err = repo.Create(ctx, article2)
	if err != nil {
		t.Fatalf("failed to create duplicate article: %v", err)
	}

	// Verify the article was updated (not duplicated)
	retrieved, err := repo.GetByID(ctx, article1.ID)
	if err != nil {
		t.Fatalf("failed to retrieve article: %v", err)
	}

	// Content should be updated
	if retrieved.Title != "Updated Title" {
		t.Errorf("expected title to be updated to 'Updated Title', got %s", retrieved.Title)
	}
	if retrieved.Description != "Updated description" {
		t.Errorf("expected description to be updated, got %s", retrieved.Description)
	}
	if retrieved.Content != "Updated content" {
		t.Errorf("expected content to be updated, got %s", retrieved.Content)
	}

	// Engagement metrics should be preserved
	if retrieved.Upvotes != 5 {
		t.Errorf("expected upvotes to be preserved as 5, got %d", retrieved.Upvotes)
	}
	if retrieved.Downvotes != 2 {
		t.Errorf("expected downvotes to be preserved as 2, got %d", retrieved.Downvotes)
	}
	if retrieved.Recommends != 10 {
		t.Errorf("expected recommends to be preserved as 10, got %d", retrieved.Recommends)
	}

	// Verify the second ID was not created
	_, err = repo.GetByID(ctx, article2.ID)
	if err == nil {
		t.Errorf("expected article with ID %s to not exist, but it does", article2.ID)
	}

	// Verify updated_at was updated
	if retrieved.UpdatedAt.Before(retrieved.CreatedAt) {
		t.Errorf("expected updated_at to be after created_at")
	}
}

// TestCreate_DuplicateLink_DeletedArticle_NotUpdated tests that deleted articles are not updated
func TestCreate_DuplicateLink_DeletedArticle_NotUpdated(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	// Create initial article
	article1 := createTestArticle("test-id-1", "https://example.com/article1", "Original Title")
	err := repo.Create(ctx, article1)
	if err != nil {
		t.Fatalf("failed to create initial article: %v", err)
	}

	// Mark the article as deleted
	_, err = db.Exec("UPDATE articles SET deleted = true WHERE id = $1", article1.ID)
	if err != nil {
		t.Fatalf("failed to mark article as deleted: %v", err)
	}

	// Try to create duplicate article with same link
	article2 := createTestArticle("test-id-2", "https://example.com/article1", "Updated Title")
	err = repo.Create(ctx, article2)
	if err != nil {
		t.Fatalf("failed to create duplicate article: %v", err)
	}

	// Verify the deleted article was NOT updated
	retrieved, err := repo.GetByID(ctx, article1.ID)
	if err != nil {
		t.Fatalf("failed to retrieve article: %v", err)
	}

	// Title should NOT be updated (still original)
	if retrieved.Title != "Original Title" {
		t.Errorf("expected title to remain 'Original Title', got %s", retrieved.Title)
	}

	// Article should still be deleted
	if !retrieved.Deleted {
		t.Errorf("expected article to remain deleted")
	}
}

// TestCreateBatch_NewArticles tests batch insertion of new articles
func TestCreateBatch_NewArticles(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	articles := []models.Article{
		createTestArticle("test-id-1", "https://example.com/article1", "Article 1"),
		createTestArticle("test-id-2", "https://example.com/article2", "Article 2"),
		createTestArticle("test-id-3", "https://example.com/article3", "Article 3"),
	}

	err := repo.CreateBatch(ctx, articles)
	if err != nil {
		t.Fatalf("failed to create batch: %v", err)
	}

	// Verify all articles were created
	for _, article := range articles {
		retrieved, err := repo.GetByID(ctx, article.ID)
		if err != nil {
			t.Errorf("failed to retrieve article %s: %v", article.ID, err)
			continue
		}
		if retrieved.Title != article.Title {
			t.Errorf("expected title %s, got %s", article.Title, retrieved.Title)
		}
	}
}

// TestCreateBatch_WithDuplicates tests batch insertion with duplicates
func TestCreateBatch_WithDuplicates(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	// Create initial article
	article1 := createTestArticle("test-id-1", "https://example.com/article1", "Original Title")
	err := repo.Create(ctx, article1)
	if err != nil {
		t.Fatalf("failed to create initial article: %v", err)
	}

	// Set engagement metrics
	_, err = db.Exec("UPDATE articles SET upvotes = 10, downvotes = 3, recommends = 20 WHERE id = $1", article1.ID)
	if err != nil {
		t.Fatalf("failed to update metrics: %v", err)
	}

	// Create batch with one duplicate link and one new article
	articles := []models.Article{
		createTestArticle("test-id-2", "https://example.com/article1", "Updated Title"), // Duplicate link
		createTestArticle("test-id-3", "https://example.com/article2", "New Article"),    // New article
	}

	err = repo.CreateBatch(ctx, articles)
	if err != nil {
		t.Fatalf("failed to create batch: %v", err)
	}

	// Verify the duplicate was updated
	retrieved1, err := repo.GetByID(ctx, article1.ID)
	if err != nil {
		t.Fatalf("failed to retrieve article 1: %v", err)
	}

	if retrieved1.Title != "Updated Title" {
		t.Errorf("expected title to be updated, got %s", retrieved1.Title)
	}
	if retrieved1.Upvotes != 10 {
		t.Errorf("expected upvotes to be preserved, got %d", retrieved1.Upvotes)
	}

	// Verify the new article was created
	retrieved2, err := repo.GetByID(ctx, "test-id-3")
	if err != nil {
		t.Fatalf("failed to retrieve article 3: %v", err)
	}

	if retrieved2.Title != "New Article" {
		t.Errorf("expected title 'New Article', got %s", retrieved2.Title)
	}
}

// TestCreateBatch_EmptySlice tests batch insertion with empty slice
func TestCreateBatch_EmptySlice(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	err := repo.CreateBatch(ctx, []models.Article{})
	if err != nil {
		t.Errorf("expected no error for empty slice, got %v", err)
	}
}

// TestGetRecent tests retrieving recent articles
func TestGetRecent(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	// Create articles with different published times
	now := time.Now()
	articles := []models.Article{
		createTestArticle("test-id-1", "https://example.com/article1", "Article 1"),
		createTestArticle("test-id-2", "https://example.com/article2", "Article 2"),
		createTestArticle("test-id-3", "https://example.com/article3", "Article 3"),
	}
	articles[0].Published = now.Add(-3 * time.Hour)
	articles[1].Published = now.Add(-2 * time.Hour)
	articles[2].Published = now.Add(-1 * time.Hour)

	for _, article := range articles {
		err := repo.Create(ctx, article)
		if err != nil {
			t.Fatalf("failed to create article: %v", err)
		}
	}

	// Get recent articles
	recent, err := repo.GetRecent(ctx, 2)
	if err != nil {
		t.Fatalf("failed to get recent articles: %v", err)
	}

	if len(recent) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(recent))
	}

	// Should be sorted by published DESC (most recent first)
	if recent[0].ID != "test-id-3" {
		t.Errorf("expected first article to be test-id-3, got %s", recent[0].ID)
	}
	if recent[1].ID != "test-id-2" {
		t.Errorf("expected second article to be test-id-2, got %s", recent[1].ID)
	}
}

// TestGetByID_NotFound tests retrieving a non-existent article
func TestGetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := NewArticleRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent article, got nil")
	}
}
