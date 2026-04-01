//go:build integration

package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/testutil"
	"github.com/mmcdole/gofeed"
)

// Mock recommender client for testing
type mockRecommenderClient struct {
	submitted []models.Article
	shouldErr bool
}

func (m *mockRecommenderClient) SubmitArticles(ctx context.Context, articles []models.Article) error {
	if m.shouldErr {
		return fmt.Errorf("mock error")
	}
	m.submitted = append(m.submitted, articles...)
	return nil
}

func TestFilterNewArticles_FirstFetch(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)

	// Create test items with different publish times
	now := time.Now()
	items := []*gofeed.Item{
		{
			Title:           "Article 1",
			Link:            "https://example.com/article1",
			Description:     "Description 1",
			PublishedParsed: &now,
		},
		{
			Title:           "Article 2",
			Link:            "https://example.com/article2",
			Description:     "Description 2",
			PublishedParsed: ptrTime(now.Add(-1 * time.Hour)),
		},
	}

	feed := &gofeed.Feed{
		Title: "Test Feed",
	}

	// Filter with nil lastFetch (first fetch) - should return all articles
	articles := fetcher.filterNewArticles(items, nil, feed, "https://example.com/feed.xml")

	if len(articles) != 2 {
		t.Errorf("Expected 2 articles on first fetch, got %d", len(articles))
	}
}

func TestFilterNewArticles_OnlyNew(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)

	// Last fetch was 2 hours ago
	lastFetch := time.Now().Add(-2 * time.Hour)

	// Create test items: one new (1 hour ago), one old (3 hours ago)
	now := time.Now()
	items := []*gofeed.Item{
		{
			Title:           "New Article",
			Link:            "https://example.com/new",
			Description:     "New article description",
			PublishedParsed: ptrTime(now.Add(-1 * time.Hour)), // After lastFetch
		},
		{
			Title:           "Old Article",
			Link:            "https://example.com/old",
			Description:     "Old article description",
			PublishedParsed: ptrTime(now.Add(-3 * time.Hour)), // Before lastFetch
		},
	}

	feed := &gofeed.Feed{
		Title: "Test Feed",
	}

	// Filter - should return only the new article
	articles := fetcher.filterNewArticles(items, &lastFetch, feed, "https://example.com/feed.xml")

	if len(articles) != 1 {
		t.Errorf("Expected 1 new article, got %d", len(articles))
	}

	if len(articles) > 0 && articles[0].Title != "New Article" {
		t.Errorf("Expected 'New Article', got '%s'", articles[0].Title)
	}
}

func TestFilterNewArticles_UseUpdatedDate(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)

	lastFetch := time.Now().Add(-2 * time.Hour)
	now := time.Now()

	// Create item without PublishedParsed but with UpdatedParsed
	items := []*gofeed.Item{
		{
			Title:         "Updated Article",
			Link:          "https://example.com/updated",
			Description:   "Updated article description",
			UpdatedParsed: ptrTime(now.Add(-1 * time.Hour)), // After lastFetch
		},
	}

	feed := &gofeed.Feed{
		Title: "Test Feed",
	}

	// Filter - should use UpdatedParsed when PublishedParsed is nil
	articles := fetcher.filterNewArticles(items, &lastFetch, feed, "https://example.com/feed.xml")

	if len(articles) != 1 {
		t.Errorf("Expected 1 article (using UpdatedParsed), got %d", len(articles))
	}
}

func TestFilterNewArticles_EmptyItems(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)

	lastFetch := time.Now().Add(-2 * time.Hour)
	items := []*gofeed.Item{}

	feed := &gofeed.Feed{
		Title: "Test Feed",
	}

	// Filter empty items
	articles := fetcher.filterNewArticles(items, &lastFetch, feed, "https://example.com/feed.xml")

	if len(articles) != 0 {
		t.Errorf("Expected 0 articles from empty items, got %d", len(articles))
	}
}

func TestConvertToArticle_Complete(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)

	publishedTime := time.Now()
	item := &gofeed.Item{
		Title:           "Test Article",
		Link:            "https://example.com/article",
		Description:     "Test description",
		Content:         "Full article content",
		PublishedParsed: &publishedTime,
		Author:          &gofeed.Person{Name: "John Doe"},
		Categories:      []string{"Tech", "News"},
	}

	feed := &gofeed.Feed{
		Title: "Test Feed",
	}

	article := fetcher.convertToArticle(item, feed, "https://example.com/feed.xml")

	if article.Title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got '%s'", article.Title)
	}

	if article.Link != "https://example.com/article" {
		t.Errorf("Expected link 'https://example.com/article', got '%s'", article.Link)
	}

	if article.Content != "Full article content" {
		t.Errorf("Expected content 'Full article content', got '%s'", article.Content)
	}

	if article.Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got '%s'", article.Author)
	}

	if len(article.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(article.Categories))
	}

	if article.FeedURL != "https://example.com/feed.xml" {
		t.Errorf("Expected FeedURL 'https://example.com/feed.xml', got '%s'", article.FeedURL)
	}

	if article.FeedTitle != "Test Feed" {
		t.Errorf("Expected FeedTitle 'Test Feed', got '%s'", article.FeedTitle)
	}

	// Verify ID is generated
	if article.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestConvertToArticle_Minimal(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)

	// Minimal item with only required fields
	item := &gofeed.Item{
		Title:       "Minimal Article",
		Link:        "https://example.com/minimal",
		Description: "Description only",
	}

	feed := &gofeed.Feed{
		Title: "Test Feed",
	}

	article := fetcher.convertToArticle(item, feed, "https://example.com/feed.xml")

	if article.Title != "Minimal Article" {
		t.Errorf("Expected title 'Minimal Article', got '%s'", article.Title)
	}

	// Content should fallback to Description
	if article.Content != "Description only" {
		t.Errorf("Expected content to fallback to description, got '%s'", article.Content)
	}

	if article.Author != "" {
		t.Errorf("Expected empty author, got '%s'", article.Author)
	}

	if len(article.Categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(article.Categories))
	}
}

func TestGenerateID_Consistency(t *testing.T) {
	// Same link should always generate same ID
	link := "https://example.com/article"

	id1 := generateID(link)
	id2 := generateID(link)

	if id1 != id2 {
		t.Error("Expected same ID for same link")
	}

	// Different links should generate different IDs
	link2 := "https://example.com/different"
	id3 := generateID(link2)

	if id1 == id3 {
		t.Error("Expected different IDs for different links")
	}
}

func TestFetchSingleFeed_Success(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	// Create a test HTTP server that serves RSS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 3)))
	}))
	defer server.Close()

	// Create a feed in the database with the test server URL
	feedID := testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	// Create fetcher and fetch
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := context.Background()

	err := fetcher.FetchSingleFeed(ctx)
	if err != nil {
		t.Fatalf("FetchSingleFeed failed: %v", err)
	}

	// Verify articles were submitted to recommender
	if len(mockClient.submitted) != 3 {
		t.Errorf("Expected 3 articles submitted, got %d", len(mockClient.submitted))
	}

	// Verify feed was updated
	_, _, failures, lastFetched := testutil.GetFeedByID(t, database, feedID)
	if failures != 0 {
		t.Errorf("Expected consecutive_failures=0, got %d", failures)
	}
	if lastFetched == nil {
		t.Error("Expected last_fetched_at to be set")
	}

	// Verify fetch history was recorded
	historyCount := testutil.CountFetchHistory(t, database)
	if historyCount != 1 {
		t.Errorf("Expected 1 fetch history record, got %d", historyCount)
	}
}

func TestFetchSingleFeed_HTTPError(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	// Create a test HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a feed in the database with the test server URL
	feedID := testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	// Create fetcher and fetch
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := context.Background()

	err := fetcher.FetchSingleFeed(ctx)
	if err == nil {
		t.Fatal("Expected error when HTTP server returns 500, got nil")
	}

	// Verify no articles were submitted
	if len(mockClient.submitted) != 0 {
		t.Errorf("Expected 0 articles submitted on error, got %d", len(mockClient.submitted))
	}

	// Verify feed failure was tracked
	_, _, failures, _ := testutil.GetFeedByID(t, database, feedID)
	if failures != 1 {
		t.Errorf("Expected consecutive_failures=1, got %d", failures)
	}

	// Verify fetch history was recorded
	historyCount := testutil.CountFetchHistory(t, database)
	if historyCount != 1 {
		t.Errorf("Expected 1 fetch history record, got %d", historyCount)
	}
}

func TestFetchSingleFeed_RecommenderError(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{shouldErr: true}

	// Create a test HTTP server that serves RSS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 2)))
	}))
	defer server.Close()

	// Create a feed in the database
	feedID := testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	// Create fetcher and fetch
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := context.Background()

	err := fetcher.FetchSingleFeed(ctx)
	if err == nil {
		t.Fatal("Expected error when recommender client fails, got nil")
	}

	// Verify feed failure was tracked
	_, _, failures, _ := testutil.GetFeedByID(t, database, feedID)
	if failures != 1 {
		t.Errorf("Expected consecutive_failures=1, got %d", failures)
	}
}

func TestFetchSingleFeed_NoEnabledFeeds(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	// Create only disabled feeds
	testutil.CreateTestFeed(t, database, testutil.TestFeedURL1, false, nil, 10)

	// Create fetcher and fetch
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := context.Background()

	err := fetcher.FetchSingleFeed(ctx)
	if err == nil {
		t.Fatal("Expected error when no enabled feeds exist, got nil")
	}

	// Verify no articles were submitted
	if len(mockClient.submitted) != 0 {
		t.Errorf("Expected 0 articles submitted, got %d", len(mockClient.submitted))
	}
}

func TestFetchSingleFeed_OnlyNewArticles(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	// Create a test HTTP server that serves RSS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 5)))
	}))
	defer server.Close()

	// Create a feed that was last fetched 2 hours ago
	lastFetch := time.Now().Add(-2 * time.Hour)
	feedID := testutil.CreateTestFeed(t, database, server.URL, true, &lastFetch, 0)

	// Create fetcher and fetch
	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := context.Background()

	err := fetcher.FetchSingleFeed(ctx)
	if err != nil {
		t.Fatalf("FetchSingleFeed failed: %v", err)
	}

	// Since our sample RSS feed creates articles with publish times in the past,
	// and we last fetched 2 hours ago, articles published in the last 1 hour
	// should be submitted (based on testutil.SampleRSSFeed implementation)
	// The sample RSS feed creates articles 1, 2, 3, 4, 5 hours ago
	// Only the article from 1 hour ago should be new
	if len(mockClient.submitted) != 1 {
		t.Logf("Articles submitted: %d", len(mockClient.submitted))
		t.Errorf("Expected 1 new article (published in last 2 hours), got %d", len(mockClient.submitted))
	}

	// Verify feed was updated
	_, _, failures, lastFetchedAfter := testutil.GetFeedByID(t, database, feedID)
	if failures != 0 {
		t.Errorf("Expected consecutive_failures=0, got %d", failures)
	}
	if lastFetchedAfter == nil {
		t.Fatal("Expected last_fetched_at to be updated")
	}
	if !lastFetchedAfter.After(lastFetch) {
		t.Error("Expected last_fetched_at to be updated to a more recent time")
	}
}

// Helper function to create time pointers
func ptrTime(t time.Time) *time.Time {
	return &t
}

// Compile-time check that mockRecommenderClient implements client.RecommenderClient interface
var _ interface {
	SubmitArticles(ctx context.Context, articles []models.Article) error
} = (*mockRecommenderClient)(nil)
