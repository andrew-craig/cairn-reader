//go:build integration

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/fetcher"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/sync"
	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/testutil"
)

// TestEndToEndFlow tests the complete fetcher workflow:
// 1. Sync feeds from Kagi list
// 2. Fetch feeds one at a time
// 3. Submit articles to recommender
// 4. Track feed health
func TestEndToEndFlow(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	ctx := context.Background()

	// Create mock Kagi feed list server
	kagiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`https://example.com/feed1.xml
https://example.com/feed2.xml
https://example.com/feed3.xml`))
	}))
	defer kagiServer.Close()

	// Step 1: Sync feeds from Kagi list
	syncer := sync.NewFeedSyncer(repo, kagiServer.URL)
	err := syncer.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}

	// Verify feeds were imported
	count := testutil.CountFeeds(t, database)
	if count != 3 {
		t.Fatalf("Expected 3 feeds after sync, got %d", count)
	}

	// Step 2: Update feed URLs to point to test RSS servers
	// Create RSS feed servers
	rssServer1 := createRSSServer(t, "Feed 1", 5)
	defer rssServer1.Close()

	rssServer2 := createRSSServer(t, "Feed 2", 3)
	defer rssServer2.Close()

	rssServer3 := createRSSServer(t, "Feed 3", 2)
	defer rssServer3.Close()

	// Update feed URLs in database to use test servers
	_, err = database.Exec(ctx, "UPDATE feeds SET url = $1 WHERE id = 1", rssServer1.URL)
	if err != nil {
		t.Fatalf("Failed to update feed 1: %v", err)
	}
	_, err = database.Exec(ctx, "UPDATE feeds SET url = $1 WHERE id = 2", rssServer2.URL)
	if err != nil {
		t.Fatalf("Failed to update feed 2: %v", err)
	}
	_, err = database.Exec(ctx, "UPDATE feeds SET url = $1 WHERE id = 3", rssServer3.URL)
	if err != nil {
		t.Fatalf("Failed to update feed 3: %v", err)
	}

	// Step 3: Create mock recommender server
	articleCount := 0
	recommenderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/explore/articles" && r.Method == http.MethodPost {
			articleCount++
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"status":"success"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer recommenderServer.Close()

	recommenderClient := client.NewRecommenderClient(recommenderServer.URL)

	// Step 4: Fetch feeds one by one
	f := fetcher.NewFetcher(repo, recommenderClient, 60*time.Second)

	// Fetch first feed (should be feed 1 since none have been fetched)
	err = f.FetchSingleFeed(ctx)
	if err != nil {
		t.Fatalf("First fetch failed: %v", err)
	}

	// Verify articles were submitted
	if articleCount != 1 {
		t.Errorf("Expected 1 article submission after first fetch, got %d", articleCount)
	}

	// Verify feed 1 was updated
	feed1, err := repo.GetFeedByID(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get feed 1: %v", err)
	}
	if feed1.LastFetchedAt == nil {
		t.Error("Expected feed 1 last_fetched_at to be set")
	}
	if feed1.ConsecutiveFailures != 0 {
		t.Errorf("Expected feed 1 consecutive_failures=0, got %d", feed1.ConsecutiveFailures)
	}

	// Fetch second feed
	err = f.FetchSingleFeed(ctx)
	if err != nil {
		t.Fatalf("Second fetch failed: %v", err)
	}

	if articleCount != 2 {
		t.Errorf("Expected 2 article submissions after second fetch, got %d", articleCount)
	}

	// Fetch third feed
	err = f.FetchSingleFeed(ctx)
	if err != nil {
		t.Fatalf("Third fetch failed: %v", err)
	}

	if articleCount != 3 {
		t.Errorf("Expected 3 article submissions after third fetch, got %d", articleCount)
	}

	// Verify all feeds were fetched
	feeds, err := repo.ListFeeds(ctx, false)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}

	for _, feed := range feeds {
		if feed.LastFetchedAt == nil {
			t.Errorf("Expected feed %d to be fetched", feed.ID)
		}
		if feed.ConsecutiveFailures != 0 {
			t.Errorf("Expected feed %d consecutive_failures=0, got %d", feed.ID, feed.ConsecutiveFailures)
		}
	}

	// Verify fetch history was recorded
	historyCount := testutil.CountFetchHistory(t, database)
	if historyCount != 3 {
		t.Errorf("Expected 3 fetch history records, got %d", historyCount)
	}
}

// TestFeedHealthTracking tests the feed health tracking and auto-disable after 10 failures
func TestFeedHealthTracking(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	ctx := context.Background()

	// Create a failing RSS server
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failingServer.Close()

	// Create a feed
	feedID := testutil.CreateTestFeed(t, database, failingServer.URL, true, nil, 0)

	// Create mock recommender client
	recommenderServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer recommenderServer.Close()
	recommenderClient := client.NewRecommenderClient(recommenderServer.URL)

	// Create fetcher
	f := fetcher.NewFetcher(repo, recommenderClient, 60*time.Second)

	// Fetch 10 times (should fail each time and disable feed)
	for i := 1; i <= 10; i++ {
		err := f.FetchSingleFeed(ctx)
		if err == nil {
			t.Fatalf("Expected error on fetch %d, got nil", i)
		}

		// Check consecutive failures
		_, enabled, failures, _ := testutil.GetFeedByID(t, database, feedID)

		if i < 10 {
			if failures != i {
				t.Errorf("After fetch %d, expected consecutive_failures=%d, got %d", i, i, failures)
			}
			if !enabled {
				t.Errorf("Expected feed to remain enabled after %d failures", i)
			}
		} else {
			// After 10th failure, feed should be disabled
			if failures != 10 {
				t.Errorf("Expected consecutive_failures=10, got %d", failures)
			}
			if enabled {
				t.Error("Expected feed to be disabled after 10 failures")
			}
		}
	}

	// Verify fetch history was recorded for all failures
	historyCount := testutil.CountFetchHistory(t, database)
	if historyCount != 10 {
		t.Errorf("Expected 10 fetch history records, got %d", historyCount)
	}
}

// TestFeedPrioritization tests that never-fetched feeds are prioritized over old fetches
func TestFeedPrioritization(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	ctx := context.Background()

	// Create feeds: one old fetch, one newer fetch, one never fetched
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	oldFeedID := testutil.CreateTestFeed(t, database, "https://example.com/old.xml", true, &weekAgo, 0)
	newFeedID := testutil.CreateTestFeed(t, database, "https://example.com/new.xml", true, &yesterday, 0)
	neverFetchedID := testutil.CreateTestFeed(t, database, "https://example.com/never.xml", true, nil, 0)

	// GetNextFeed should return never-fetched feed first
	feed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed.ID != neverFetchedID {
		t.Errorf("Expected never-fetched feed (ID %d), got ID %d", neverFetchedID, feed.ID)
	}

	// Mark never-fetched feed as fetched
	err = repo.UpdateFetchResult(ctx, neverFetchedID, true)
	if err != nil {
		t.Fatalf("UpdateFetchResult failed: %v", err)
	}

	// GetNextFeed should now return oldest-fetched feed
	feed, err = repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed.ID != oldFeedID {
		t.Errorf("Expected oldest feed (ID %d), got ID %d", oldFeedID, feed.ID)
	}

	// Mark old feed as fetched
	err = repo.UpdateFetchResult(ctx, oldFeedID, true)
	if err != nil {
		t.Fatalf("UpdateFetchResult failed: %v", err)
	}

	// GetNextFeed should now return the remaining feed
	feed, err = repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed.ID != newFeedID {
		t.Errorf("Expected newer feed (ID %d), got ID %d", newFeedID, feed.ID)
	}
}

// TestDailySyncPreservesMetadata tests that daily feed sync doesn't overwrite existing metadata
func TestDailySyncPreservesMetadata(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	ctx := context.Background()

	// Create an existing feed with custom metadata
	now := time.Now()
	feedID := testutil.CreateTestFeed(t, database, "https://example.com/feed1.xml", false, &now, 5)

	// Create mock Kagi server with same feed
	kagiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("https://example.com/feed1.xml\nhttps://example.com/feed2.xml"))
	}))
	defer kagiServer.Close()

	// Sync feeds
	syncer := sync.NewFeedSyncer(repo, kagiServer.URL)
	err := syncer.SyncOnce(ctx)
	if err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}

	// Verify existing feed metadata was preserved
	_, enabled, failures, lastFetched := testutil.GetFeedByID(t, database, feedID)
	if enabled {
		t.Error("Expected existing feed to remain disabled")
	}
	if failures != 5 {
		t.Errorf("Expected existing feed to preserve consecutive_failures=5, got %d", failures)
	}
	if lastFetched == nil {
		t.Error("Expected existing feed to preserve last_fetched_at")
	}

	// Verify new feed was added
	count := testutil.CountFeeds(t, database)
	if count != 2 {
		t.Errorf("Expected 2 feeds (1 existing + 1 new), got %d", count)
	}
}

// Helper function to create a test RSS server
func createRSSServer(t *testing.T, title string, itemCount int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)

		items := ""
		for i := 1; i <= itemCount; i++ {
			items += fmt.Sprintf(`
		<item>
			<title>Article %d - %s</title>
			<link>https://example.com/%s/article-%d</link>
			<description>Description for article %d</description>
			<pubDate>%s</pubDate>
			<author>Test Author</author>
		</item>
			`, i, title, title, i, i, time.Now().Format(time.RFC1123Z))
		}

		rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>%s</title>
		<link>https://example.com</link>
		<description>Test RSS Feed</description>
		<language>en-us</language>
		<lastBuildDate>%s</lastBuildDate>
		%s
	</channel>
</rss>`, title, time.Now().Format(time.RFC1123Z), items)

		w.Write([]byte(rss))
	}))
}
