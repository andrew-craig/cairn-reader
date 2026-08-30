//go:build integration

package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/models"
	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch/fetchtest"
	"github.com/andrew-craig/cairn-reader/pkg/rss/parse"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/testutil"
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

func TestConvertToArticle_Complete(t *testing.T) {
	publishedTime := time.Now()
	item := parse.Item{
		Title:       "Test Article",
		Link:        "https://example.com/article",
		Description: "Test description",
		Content:     "Full article content",
		PublishedAt: &publishedTime,
		Author:      "John Doe",
	}

	feed := &parse.Feed{Title: "Test Feed"}

	article := convertToArticle(item, feed, "https://example.com/feed.xml")

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
	if article.FeedURL != "https://example.com/feed.xml" {
		t.Errorf("Expected FeedURL 'https://example.com/feed.xml', got '%s'", article.FeedURL)
	}
	if article.FeedTitle != "Test Feed" {
		t.Errorf("Expected FeedTitle 'Test Feed', got '%s'", article.FeedTitle)
	}
	if article.ID == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestConvertToArticle_Minimal(t *testing.T) {
	item := parse.Item{
		Title:       "Minimal Article",
		Link:        "https://example.com/minimal",
		Description: "Description only",
		Content:     "Description only",
	}
	feed := &parse.Feed{Title: "Test Feed"}

	article := convertToArticle(item, feed, "https://example.com/feed.xml")

	if article.Title != "Minimal Article" {
		t.Errorf("Expected title 'Minimal Article', got '%s'", article.Title)
	}
	if article.Content != "Description only" {
		t.Errorf("Expected content 'Description only', got '%s'", article.Content)
	}
	if article.Author != "" {
		t.Errorf("Expected empty author, got '%s'", article.Author)
	}
}

func TestConvertToArticle_NoPublishedAtIsZeroTime(t *testing.T) {
	item := parse.Item{
		Title:   "Dateless Article",
		Link:    "https://example.com/dateless",
		Content: "content",
	}
	feed := &parse.Feed{Title: "Test Feed"}

	a1 := convertToArticle(item, feed, "https://example.com/feed.xml")
	a2 := convertToArticle(item, feed, "https://example.com/feed.xml")

	if !a1.Published.IsZero() {
		t.Errorf("Expected zero published time when item has no PublishedAt, got %v", a1.Published)
	}
	if !a1.Published.Equal(a2.Published) {
		t.Error("Expected convertToArticle to be idempotent for items without PublishedAt")
	}
}

func TestConvertToArticle_IDIsContentHash(t *testing.T) {
	item := parse.Item{
		Link:    "https://example.com/article",
		Content: "some content",
	}
	feed := &parse.Feed{Title: "Test Feed"}

	a1 := convertToArticle(item, feed, "https://example.com/feed.xml")

	// Same content → same ID
	a2 := convertToArticle(item, feed, "https://example.com/feed.xml")
	if a1.ID != a2.ID {
		t.Error("Expected same ID for identical content")
	}

	// Different content → different ID
	item2 := parse.Item{
		Link:    "https://example.com/article",
		Content: "different content",
	}
	a3 := convertToArticle(item2, feed, "https://example.com/feed.xml")
	if a1.ID == a3.ID {
		t.Error("Expected different IDs for different content")
	}
}

func TestFetchSingleFeed_Success(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 3)))
	}))
	defer server.Close()

	feedID := testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := fetchtest.AllowLoopback(context.Background())

	err := fetcher.FetchSingleFeed(ctx)
	if err != nil {
		t.Fatalf("FetchSingleFeed failed: %v", err)
	}

	if len(mockClient.submitted) != 3 {
		t.Errorf("Expected 3 articles submitted, got %d", len(mockClient.submitted))
	}

	_, _, failures, lastFetched := testutil.GetFeedByID(t, database, feedID)
	if failures != 0 {
		t.Errorf("Expected consecutive_failures=0, got %d", failures)
	}
	if lastFetched == nil {
		t.Error("Expected last_fetched_at to be set")
	}

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	feedID := testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := fetchtest.AllowLoopback(context.Background())

	err := fetcher.FetchSingleFeed(ctx)
	if err == nil {
		t.Fatal("Expected error when HTTP server returns 500, got nil")
	}

	if len(mockClient.submitted) != 0 {
		t.Errorf("Expected 0 articles submitted on error, got %d", len(mockClient.submitted))
	}

	_, _, failures, _ := testutil.GetFeedByID(t, database, feedID)
	if failures != 1 {
		t.Errorf("Expected consecutive_failures=1, got %d", failures)
	}

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 2)))
	}))
	defer server.Close()

	feedID := testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := fetchtest.AllowLoopback(context.Background())

	err := fetcher.FetchSingleFeed(ctx)
	if err == nil {
		t.Fatal("Expected error when recommender client fails, got nil")
	}

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

	testutil.CreateTestFeed(t, database, testutil.TestFeedURL1, false, nil, 10)

	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := fetchtest.AllowLoopback(context.Background())

	err := fetcher.FetchSingleFeed(ctx)
	if err == nil {
		t.Fatal("Expected error when no enabled feeds exist, got nil")
	}

	if len(mockClient.submitted) != 0 {
		t.Errorf("Expected 0 articles submitted, got %d", len(mockClient.submitted))
	}
}

func TestFetchSingleFeed_ForwardsAllItems(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 5)))
	}))
	defer server.Close()

	lastFetch := time.Now().Add(-2 * time.Hour)
	feedID := testutil.CreateTestFeed(t, database, server.URL, true, &lastFetch, 0)

	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := fetchtest.AllowLoopback(context.Background())

	if err := fetcher.FetchSingleFeed(ctx); err != nil {
		t.Fatalf("FetchSingleFeed failed: %v", err)
	}

	// All items are forwarded regardless of last_fetched_at; the recommender
	// deduplicates by link.
	if len(mockClient.submitted) != 5 {
		t.Errorf("Expected 5 articles submitted, got %d", len(mockClient.submitted))
	}

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

func TestFetchSingleFeed_ETagSentOnSubsequentFetch(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	mockClient := &mockRecommenderClient{}

	const testETag = `"abc123"`
	requestCount := 0
	var receivedIfNoneMatch string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		receivedIfNoneMatch = r.Header.Get("If-None-Match")

		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("ETag", testETag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testutil.SampleRSSFeed("Test Feed", 1)))
	}))
	defer server.Close()

	testutil.CreateTestFeed(t, database, server.URL, true, nil, 0)

	fetcher := NewFetcher(repo, mockClient, 60*time.Second)
	ctx := fetchtest.AllowLoopback(context.Background())

	// First fetch — no conditional headers sent.
	if err := fetcher.FetchSingleFeed(ctx); err != nil {
		t.Fatalf("first FetchSingleFeed failed: %v", err)
	}
	if receivedIfNoneMatch != "" {
		t.Errorf("Expected no If-None-Match on first fetch, got %q", receivedIfNoneMatch)
	}

	// Second fetch — ETag from first response must be sent as If-None-Match.
	if err := fetcher.FetchSingleFeed(ctx); err != nil {
		t.Fatalf("second FetchSingleFeed failed: %v", err)
	}
	if receivedIfNoneMatch != testETag {
		t.Errorf("Expected If-None-Match=%q on second fetch, got %q", testETag, receivedIfNoneMatch)
	}
}

// Compile-time check that mockRecommenderClient implements client.RecommenderClient interface
var _ interface {
	SubmitArticles(ctx context.Context, articles []models.Article) error
} = (*mockRecommenderClient)(nil)
