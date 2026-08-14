//go:build integration
// +build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// cleanupTestData removes all test data from the database
func cleanupTestData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	queries := []string{
		"DELETE FROM user_article_recommendations WHERE article_id LIKE 'test-%'",
		"DELETE FROM user_articles WHERE article_id LIKE 'test-%'",
		"DELETE FROM articles WHERE id LIKE 'test-%'",
		"DELETE FROM users WHERE user_id LIKE 'test-%'",
	}

	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Logf("Warning: failed to clean up: %v", err)
		}
	}
}

// createIntegrationTestArticle creates a test article with default values
func createIntegrationTestArticle(id, link, title string) models.Article {
	return models.Article{
		ID:          id,
		Title:       title,
		Link:        link,
		Description: "Test description",
		Content:     "Test content for integration testing",
		Author:      "Test Author",
		Published:   time.Now(),
		FeedURL:     "https://example.com/feed",
		FeedTitle:   "Test Feed",
		Categories:  []string{"tech", "programming"},
	}
}

// TestIntegration_Create_NewArticle tests inserting a new article
func TestIntegration_Create_NewArticle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer cleanupTestData(t, pool)

	repo := NewArticleRepository(pool)
	ctx := context.Background()

	article := createIntegrationTestArticle("test-new-article-1", "https://example.com/test/article1", "Test Article 1")

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

// TestIntegration_Create_DuplicateLink_UpdatesArticle tests Phase 2 deduplication
func TestIntegration_Create_DuplicateLink_UpdatesArticle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer cleanupTestData(t, pool)

	repo := NewArticleRepository(pool)
	ctx := context.Background()

	// Create initial article
	article1 := createIntegrationTestArticle("test-dup-1", "https://example.com/test/duplicate", "Original Title")
	err := repo.Create(ctx, article1)
	if err != nil {
		t.Fatalf("failed to create initial article: %v", err)
	}

	// Manually set some engagement metrics
	_, err = pool.Exec(ctx, "UPDATE articles SET upvotes = 5, downvotes = 2, recommends = 10 WHERE id = $1", article1.ID)
	if err != nil {
		t.Fatalf("failed to update metrics: %v", err)
	}

	// Sleep briefly to ensure updated_at will be different
	time.Sleep(100 * time.Millisecond)

	// Create duplicate article with same link but different ID and content
	article2 := createIntegrationTestArticle("test-dup-2", "https://example.com/test/duplicate", "Updated Title")
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

	// Engagement metrics should be preserved (Phase 2 requirement)
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
	if !retrieved.UpdatedAt.After(retrieved.CreatedAt) {
		t.Errorf("expected updated_at (%v) to be after created_at (%v)", retrieved.UpdatedAt, retrieved.CreatedAt)
	}
}

// TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated tests Phase 2 requirement:
// Don't update deleted articles
func TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer cleanupTestData(t, pool)

	repo := NewArticleRepository(pool)
	ctx := context.Background()

	// Create initial article
	article1 := createIntegrationTestArticle("test-deleted-1", "https://example.com/test/deleted", "Original Title")
	err := repo.Create(ctx, article1)
	if err != nil {
		t.Fatalf("failed to create initial article: %v", err)
	}

	// Mark the article as deleted
	_, err = pool.Exec(ctx, "UPDATE articles SET deleted = true WHERE id = $1", article1.ID)
	if err != nil {
		t.Fatalf("failed to mark article as deleted: %v", err)
	}

	// Get original updated_at timestamp
	var originalUpdatedAt time.Time
	err = pool.QueryRow(ctx, "SELECT updated_at FROM articles WHERE id = $1", article1.ID).Scan(&originalUpdatedAt)
	if err != nil {
		t.Fatalf("failed to get original updated_at: %v", err)
	}

	// Sleep to ensure timestamps would be different
	time.Sleep(100 * time.Millisecond)

	// Try to create duplicate article with same link
	article2 := createIntegrationTestArticle("test-deleted-2", "https://example.com/test/deleted", "Updated Title")
	article2.Description = "Updated description"
	err = repo.Create(ctx, article2)
	if err != nil {
		t.Fatalf("failed to create duplicate article: %v", err)
	}

	// Verify the deleted article was NOT updated (Phase 2 requirement)
	retrieved, err := repo.GetByID(ctx, article1.ID)
	if err != nil {
		t.Fatalf("failed to retrieve article: %v", err)
	}

	// Title should NOT be updated (still original)
	if retrieved.Title != "Original Title" {
		t.Errorf("expected title to remain 'Original Title', got %s", retrieved.Title)
	}

	// Description should NOT be updated
	if retrieved.Description != "Test description" {
		t.Errorf("expected description to remain 'Test description', got %s", retrieved.Description)
	}

	// Article should still be deleted
	if !retrieved.Deleted {
		t.Errorf("expected article to remain deleted")
	}

	// updated_at should NOT have changed
	if !retrieved.UpdatedAt.Equal(originalUpdatedAt) {
		t.Errorf("expected updated_at to remain unchanged, got %v, want %v", retrieved.UpdatedAt, originalUpdatedAt)
	}
}

// TestIntegration_CreateBatch_WithDuplicates tests batch insertion with deduplication
func TestIntegration_CreateBatch_WithDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer cleanupTestData(t, pool)

	repo := NewArticleRepository(pool)
	ctx := context.Background()

	// Create initial article
	article1 := createIntegrationTestArticle("test-batch-1", "https://example.com/test/batch1", "Original Title")
	err := repo.Create(ctx, article1)
	if err != nil {
		t.Fatalf("failed to create initial article: %v", err)
	}

	// Set engagement metrics
	_, err = pool.Exec(ctx, "UPDATE articles SET upvotes = 10, downvotes = 3, recommends = 20 WHERE id = $1", article1.ID)
	if err != nil {
		t.Fatalf("failed to update metrics: %v", err)
	}

	// Create batch with one duplicate link and two new articles
	articles := []models.Article{
		createIntegrationTestArticle("test-batch-2", "https://example.com/test/batch1", "Updated Title"), // Duplicate link
		createIntegrationTestArticle("test-batch-3", "https://example.com/test/batch2", "New Article 1"), // New article
		createIntegrationTestArticle("test-batch-4", "https://example.com/test/batch3", "New Article 2"), // New article
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
	if retrieved1.Recommends != 20 {
		t.Errorf("expected recommends to be preserved, got %d", retrieved1.Recommends)
	}

	// Verify the new articles were created
	retrieved2, err := repo.GetByID(ctx, "test-batch-3")
	if err != nil {
		t.Fatalf("failed to retrieve article 3: %v", err)
	}
	if retrieved2.Title != "New Article 1" {
		t.Errorf("expected title 'New Article 1', got %s", retrieved2.Title)
	}

	retrieved3, err := repo.GetByID(ctx, "test-batch-4")
	if err != nil {
		t.Fatalf("failed to retrieve article 4: %v", err)
	}
	if retrieved3.Title != "New Article 2" {
		t.Errorf("expected title 'New Article 2', got %s", retrieved3.Title)
	}
}

// TestIntegration_CreateBatch_ContentHashCollision_BatchSurvives covers P2-C7/H8:
// article ids are content hashes, independent of link, so syndicated content
// re-published under a new link collides on the id (primary key) rather than
// the link. Before the fix, that PK violation rolled back the whole
// multi-row INSERT and silently dropped every other article in the batch.
func TestIntegration_CreateBatch_ContentHashCollision_BatchSurvives(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer cleanupTestData(t, pool)

	repo := NewArticleRepository(pool)
	ctx := context.Background()

	// A previous poll already stored this content under one link.
	original := createIntegrationTestArticle("test-hash-collision-shared-id", "https://example.com/test/hash-collision/original", "Original Source")
	if err := repo.Create(ctx, original); err != nil {
		t.Fatalf("failed to seed original article: %v", err)
	}

	// The next poll delivers the same content syndicated under a different
	// link (shares only the id with the existing row), alongside a
	// genuinely new, unrelated article.
	syndicated := createIntegrationTestArticle("test-hash-collision-shared-id", "https://example.com/test/hash-collision/syndicated", "Syndicated Copy")
	freshArticle := createIntegrationTestArticle("test-hash-collision-new", "https://example.com/test/hash-collision/new", "Brand New Article")

	err := repo.CreateBatch(ctx, []models.Article{syndicated, freshArticle})
	if err != nil {
		t.Fatalf("CreateBatch returned an error on a content-hash collision: %v", err)
	}

	// The fresh article must not have been dropped by the collision.
	if _, err := repo.GetByID(ctx, freshArticle.ID); err != nil {
		t.Errorf("expected fresh article to survive the batch, got error: %v", err)
	}

	// Deliberate choice: the first-seen link stays canonical for a given
	// content hash; the syndicated duplicate is dropped rather than
	// overwriting it.
	retrieved, err := repo.GetByID(ctx, original.ID)
	if err != nil {
		t.Fatalf("failed to retrieve original article: %v", err)
	}
	if retrieved.Link != original.Link {
		t.Errorf("expected canonical link to remain %s, got %s", original.Link, retrieved.Link)
	}

	// No second row was created for the syndicated link.
	rows, err := pool.Query(ctx, "SELECT COUNT(*) FROM articles WHERE link = $1", syndicated.Link)
	if err != nil {
		t.Fatalf("failed to count articles by syndicated link: %v", err)
	}
	defer rows.Close()
	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			t.Fatalf("failed to scan count: %v", err)
		}
	}
	if count != 0 {
		t.Errorf("expected no article stored under the syndicated link, found %d", count)
	}
}

// TestIntegration_CreateBatch_DuplicateAndEmptyLinksWithinBatch covers the
// H8 "empty or duplicate link" batch-poisoning mode: a single multi-row
// INSERT can't target the same ON CONFLICT (link) row twice, so two
// in-batch rows sharing a link (including two empty links) previously
// failed the whole batch with "cannot affect row a second time".
func TestIntegration_CreateBatch_DuplicateAndEmptyLinksWithinBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := testutil.SetupTestDB(t)
	defer cleanup()
	defer cleanupTestData(t, pool)

	repo := NewArticleRepository(pool)
	ctx := context.Background()

	sameLinkFirst := createIntegrationTestArticle("test-batch-dup-link-1", "https://example.com/test/batch-dup/shared-link", "First Appearance")
	sameLinkSecond := createIntegrationTestArticle("test-batch-dup-link-2", "https://example.com/test/batch-dup/shared-link", "Second Appearance")
	emptyLinkFirst := createIntegrationTestArticle("test-batch-dup-empty-1", "", "No Link 1")
	emptyLinkSecond := createIntegrationTestArticle("test-batch-dup-empty-2", "", "No Link 2")
	freshArticle := createIntegrationTestArticle("test-batch-dup-fresh", "https://example.com/test/batch-dup/fresh", "Fresh Article")

	err := repo.CreateBatch(ctx, []models.Article{sameLinkFirst, sameLinkSecond, emptyLinkFirst, emptyLinkSecond, freshArticle})
	if err != nil {
		t.Fatalf("CreateBatch returned an error on in-batch duplicate/empty links: %v", err)
	}

	// The fresh article must not have been dropped.
	if _, err := repo.GetByID(ctx, freshArticle.ID); err != nil {
		t.Errorf("expected fresh article to survive the batch, got error: %v", err)
	}

	// First-seen of the shared link wins; the second is dropped.
	if _, err := repo.GetByID(ctx, sameLinkFirst.ID); err != nil {
		t.Errorf("expected first-seen duplicate-link article to be created, got error: %v", err)
	}
	if _, err := repo.GetByID(ctx, sameLinkSecond.ID); err == nil {
		t.Errorf("expected second duplicate-link article to be dropped, but it was created")
	}

	// Both empty-link articles are dropped.
	if _, err := repo.GetByID(ctx, emptyLinkFirst.ID); err == nil {
		t.Errorf("expected first empty-link article to be dropped, but it was created")
	}
	if _, err := repo.GetByID(ctx, emptyLinkSecond.ID); err == nil {
		t.Errorf("expected second empty-link article to be dropped, but it was created")
	}
}
