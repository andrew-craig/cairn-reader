package recommend

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/recommender/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Helper functions

func createTestArticle(id, title string, upvotes, downvotes, recommends int) models.Article {
	return models.Article{
		ID:          id,
		Title:       title,
		Link:        "https://example.com/article/" + id,
		Description: "Test description",
		Content:     "Test content",
		Author:      "Test Author",
		Published:   time.Now(),
		FeedURL:     "https://example.com/feed",
		FeedTitle:   "Test Feed",
		Categories:  []string{"tech"},
		Upvotes:     upvotes,
		Downvotes:   downvotes,
		Recommends:  recommends,
		Deleted:     false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	connStr := "postgres://cairn:cairn_password@localhost:5432/cairn_test?sslmode=disable"
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping test: could not ping test database: %v", err)
	}

	// Clean up test data
	cleanupTestDB(t, pool)

	cleanup := func() {
		cleanupTestDB(t, pool)
		pool.Close()
	}

	return pool, cleanup
}

// cleanupTestDB removes all test data from the database
func cleanupTestDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	queries := []string{
		"DELETE FROM user_article_recommendations",
		"DELETE FROM user_articles",
		"DELETE FROM votes",
		"DELETE FROM articles",
		"DELETE FROM users",
	}

	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			t.Logf("Warning: failed to clean up: %v", err)
		}
	}
}

// Unit Tests - Pure Functions (no database required)

func TestCalculateQualityScore(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name        string
		upvotes     int
		downvotes   int
		recommends  int
		wantScore   float64
		description string
	}{
		{
			name:        "high quality article with good ratio",
			upvotes:     10,
			downvotes:   1,
			recommends:  20,
			wantScore:   0.35, // (10 - (1*3)) / 20 = 7/20 = 0.35
			description: "Article with more upvotes than downvotes should have positive score",
		},
		{
			name:        "low quality article with many downvotes",
			upvotes:     2,
			downvotes:   5,
			recommends:  30,
			wantScore:   -0.4333333333333333, // (2 - (5*3)) / 30 = -13/30
			description: "Article with many downvotes should have negative score",
		},
		{
			name:        "no engagement - no votes",
			upvotes:     0,
			downvotes:   0,
			recommends:  10,
			wantScore:   0.0, // (0 - 0) / 10 = 0
			description: "Article with no votes should have zero score",
		},
		{
			name:        "perfect quality - only upvotes",
			upvotes:     15,
			downvotes:   0,
			recommends:  10,
			wantScore:   1.5, // 15 / 10 = 1.5
			description: "Article with only upvotes should have high positive score",
		},
		{
			name:        "new article with no recommends but upvotes",
			upvotes:     5,
			downvotes:   0,
			recommends:  0,
			wantScore:   math.Inf(1), // Positive infinity for new articles with upvotes
			description: "New article with upvotes should get infinite score to surface quickly",
		},
		{
			name:        "new article with no recommends and no votes",
			upvotes:     0,
			downvotes:   0,
			recommends:  0,
			wantScore:   1000.0, // Default high score for new articles
			description: "New article with no engagement should get default high score",
		},
		{
			name:        "heavily downvoted article",
			upvotes:     1,
			downvotes:   10,
			recommends:  50,
			wantScore:   -0.58, // (1 - (10*3)) / 50 = -29/50
			description: "Heavily downvoted article should have very negative score",
		},
		{
			name:        "balanced engagement",
			upvotes:     30,
			downvotes:   10,
			recommends:  100,
			wantScore:   0.0, // (30 - 30) / 100 = 0
			description: "When upvotes - (downvotes*3) = 0, score should be 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article := createTestArticle("test-id", "Test Article", tt.upvotes, tt.downvotes, tt.recommends)
			score := engine.calculateQualityScore(article)

			// For infinity, use special comparison
			if math.IsInf(tt.wantScore, 1) {
				if !math.IsInf(score, 1) {
					t.Errorf("calculateQualityScore() = %v, want +Inf", score)
				}
			} else if math.Abs(score-tt.wantScore) > 0.0001 {
				t.Errorf("calculateQualityScore() = %v, want %v - %s", score, tt.wantScore, tt.description)
			}
		})
	}
}

func TestSortByQuality(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name        string
		articles    []models.Article
		wantIDs     []string
		description string
	}{
		{
			name: "sorts by quality score descending",
			articles: []models.Article{
				createTestArticle("1", "Article 1", 10, 1, 20), // Score: (10-3)/20 = 0.35
				createTestArticle("2", "Article 2", 20, 0, 10), // Score: 20/10 = 2.0
				createTestArticle("3", "Article 3", 5, 5, 15),  // Score: (5-15)/15 = -0.67
				createTestArticle("4", "Article 4", 15, 2, 30), // Score: (15-6)/30 = 0.3
				createTestArticle("5", "Article 5", 8, 0, 10),  // Score: 8/10 = 0.8
			},
			wantIDs:     []string{"2", "5", "1", "4", "3"},
			description: "Articles should be ordered by descending quality score",
		},
		{
			name: "new articles with no recommends rank highest",
			articles: []models.Article{
				createTestArticle("1", "Article 1", 10, 0, 50), // Score: 10/50 = 0.2
				createTestArticle("2", "Article 2", 5, 0, 0),   // Score: Inf (new with upvotes)
				createTestArticle("3", "Article 3", 0, 0, 0),   // Score: 1000 (new no votes)
				createTestArticle("4", "Article 4", 0, 0, 10),  // Score: 0
			},
			wantIDs:     []string{"2", "3", "1", "4"},
			description: "New articles should be prioritized",
		},
		{
			name:        "empty article list",
			articles:    []models.Article{},
			wantIDs:     []string{},
			description: "Should handle empty list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine.sortByQuality(tt.articles)

			if len(tt.articles) != len(tt.wantIDs) {
				t.Fatalf("sortByQuality() produced %d articles, want %d - %s",
					len(tt.articles), len(tt.wantIDs), tt.description)
			}

			for i, article := range tt.articles {
				if article.ID != tt.wantIDs[i] {
					t.Errorf("sortByQuality()[%d] = %s, want %s - %s",
						i, article.ID, tt.wantIDs[i], tt.description)
				}
			}
		})
	}
}

// Integration Tests (require database)

func TestGetRecommendations_Integration_FiveOrFewerArticles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID

	articleRepo := db.NewArticleRepository(pool)
	userRepo := db.NewUserRepository(pool)
	engine := NewEngine(articleRepo, userRepo)

	// Create 3 test articles
	articles := []models.Article{
		createTestArticle("1", "Article 1", 10, 0, 5),
		createTestArticle("2", "Article 2", 5, 0, 3),
		createTestArticle("3", "Article 3", 8, 1, 10),
	}

	for _, article := range articles {
		if err := articleRepo.Create(ctx, article); err != nil {
			t.Fatalf("failed to create article: %v", err)
		}
	}

	recommendations, err := engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("GetRecommendations() error = %v", err)
	}

	if len(recommendations) != 3 {
		t.Errorf("Expected 3 recommendations, got %d", len(recommendations))
	}

	// GetRecommendations is a pure read after Phase B — the recommends
	// counter must NOT move until the client confirms via POST /shown.
	for _, article := range articles {
		retrieved, err := articleRepo.GetByID(ctx, article.ID)
		if err != nil {
			t.Errorf("failed to retrieve article %s: %v", article.ID, err)
			continue
		}
		if retrieved.Recommends != article.Recommends {
			t.Errorf("Article %s recommends changed unexpectedly: was %d, got %d",
				article.ID, article.Recommends, retrieved.Recommends)
		}
	}
}

func TestGetRecommendations_Integration_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID

	articleRepo := db.NewArticleRepository(pool)
	userRepo := db.NewUserRepository(pool)
	engine := NewEngine(articleRepo, userRepo)

	// Seed 15 articles with distinct quality scores. With recommends=1 the
	// score equals upvotes, so higher upvotes rank first.
	const total = 15
	for i := 0; i < total; i++ {
		article := createTestArticle(fmt.Sprintf("article%02d", i), fmt.Sprintf("Article %d", i), i, 0, 1)
		if err := articleRepo.Create(ctx, article); err != nil {
			t.Fatalf("failed to create article: %v", err)
		}
	}

	firstPage, err := engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("GetRecommendations(offset=0) error = %v", err)
	}
	if len(firstPage) != recommendationPageSize {
		t.Fatalf("expected %d articles on first page, got %d", recommendationPageSize, len(firstPage))
	}

	// First page must be ordered by descending quality score.
	for i := 1; i < len(firstPage); i++ {
		if firstPage[i-1].Upvotes < firstPage[i].Upvotes {
			t.Errorf("first page not sorted by quality: index %d (upvotes %d) before %d (upvotes %d)",
				i-1, firstPage[i-1].Upvotes, i, firstPage[i].Upvotes)
		}
	}

	secondPage, err := engine.GetRecommendations(ctx, userID, recommendationPageSize)
	if err != nil {
		t.Fatalf("GetRecommendations(offset=%d) error = %v", recommendationPageSize, err)
	}
	if len(secondPage) != total-recommendationPageSize {
		t.Fatalf("expected %d articles on second page, got %d", total-recommendationPageSize, len(secondPage))
	}

	// The two pages must be disjoint and together cover every article.
	seen := make(map[string]bool, total)
	for _, a := range firstPage {
		seen[a.ID] = true
	}
	for _, a := range secondPage {
		if seen[a.ID] {
			t.Errorf("article %s appeared on both pages", a.ID)
		}
		seen[a.ID] = true
	}
	if len(seen) != total {
		t.Errorf("expected %d distinct articles across pages, got %d", total, len(seen))
	}
}

func TestGetRecommendations_Integration_FilterDeletedArticles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440002" // Valid UUID

	articleRepo := db.NewArticleRepository(pool)
	userRepo := db.NewUserRepository(pool)
	engine := NewEngine(articleRepo, userRepo)

	// Create active articles
	activeArticles := []models.Article{
		createTestArticle("active1", "Active 1", 20, 0, 10),
		createTestArticle("active2", "Active 2", 15, 0, 10),
		createTestArticle("active3", "Active 3", 10, 0, 10),
		createTestArticle("active4", "Active 4", 8, 0, 10),
		createTestArticle("active5", "Active 5", 5, 0, 10),
	}

	for _, article := range activeArticles {
		if err := articleRepo.Create(ctx, article); err != nil {
			t.Fatalf("failed to create active article: %v", err)
		}
	}

	// Create and delete an article
	deletedArticle := createTestArticle("deleted1", "Deleted Article", 100, 0, 5)
	if err := articleRepo.Create(ctx, deletedArticle); err != nil {
		t.Fatalf("failed to create article to delete: %v", err)
	}

	// Mark as deleted
	_, err := pool.Exec(ctx, "UPDATE articles SET deleted = true WHERE id = $1", "deleted1")
	if err != nil {
		t.Fatalf("failed to mark article as deleted: %v", err)
	}

	recommendations, err := engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("GetRecommendations() error = %v", err)
	}

	// Verify deleted article is not in recommendations
	for _, rec := range recommendations {
		if rec.ID == "deleted1" {
			t.Error("Deleted article should not be in recommendations")
		}
	}

	// Verify we got 5 active articles
	if len(recommendations) != 5 {
		t.Errorf("Expected 5 recommendations (all active), got %d", len(recommendations))
	}
}

// After Phase B, GetRecommendations is a pure read: it no longer writes
// recommendations rows or increments the recommends counter. Two
// consecutive calls for the same user must therefore overlap, and the
// counter must not move. The mobile client's POST /shown call is what
// promotes "fetched" to "actually seen" — that path is exercised in the
// API-level integration tests (services/explore/recommender/integration_shown_test.go).
func TestGetRecommendations_Integration_PureRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userID := "550e8400-e29b-41d4-a716-446655440003" // Valid UUID

	articleRepo := db.NewArticleRepository(pool)
	userRepo := db.NewUserRepository(pool)
	engine := NewEngine(articleRepo, userRepo)

	for i := 1; i <= 10; i++ {
		article := createTestArticle(fmt.Sprintf("article%d", i), fmt.Sprintf("Article %d", i), 10, 0, 5)
		if err := articleRepo.Create(ctx, article); err != nil {
			t.Fatalf("failed to create article: %v", err)
		}
	}

	firstRecs, err := engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("GetRecommendations() first call error = %v", err)
	}

	secondRecs, err := engine.GetRecommendations(ctx, userID, 0)
	if err != nil {
		t.Fatalf("GetRecommendations() second call error = %v", err)
	}

	firstSet := make(map[string]bool, len(firstRecs))
	for _, rec := range firstRecs {
		firstSet[rec.ID] = true
	}
	overlap := 0
	for _, rec := range secondRecs {
		if firstSet[rec.ID] {
			overlap++
		}
	}
	if overlap == 0 {
		t.Errorf("expected overlap between consecutive recommendation calls, got none")
	}

	for _, rec := range firstRecs {
		retrieved, err := articleRepo.GetByID(ctx, rec.ID)
		if err != nil {
			t.Errorf("failed to retrieve article %s: %v", rec.ID, err)
			continue
		}
		if retrieved.Recommends != rec.Recommends {
			t.Errorf("Article %s recommends moved without /shown call: was %d, got %d",
				rec.ID, rec.Recommends, retrieved.Recommends)
		}
	}
}
