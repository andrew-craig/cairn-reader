//go:build integration

package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/db"
	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/testutil"
)

func TestSyncFeeds_Success(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)

	// Create a test HTTP server that returns sample Kagi list
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(testutil.SampleKagiList())); err != nil {
			t.Logf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	// Create syncer with test server URL
	syncer := NewFeedSyncer(repo, "", server.URL)

	// Sync feeds
	ctx := context.Background()
	err := syncer.syncFeeds(ctx)
	if err != nil {
		t.Fatalf("syncFeeds failed: %v", err)
	}

	// Verify feeds were imported
	count := testutil.CountFeeds(t, database)
	if count != 5 {
		t.Errorf("Expected 5 feeds, got %d", count)
	}

	// Verify all feeds are enabled by default
	feeds, err := repo.ListFeeds(ctx, false)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}

	for _, feed := range feeds {
		if !feed.Enabled {
			t.Errorf("Expected feed %s to be enabled", feed.URL)
		}
	}
}

func TestSyncFeeds_SkipsComments(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)

	// Create a test HTTP server with comments and blank lines
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`# Comment line
https://example.com/feed1.xml

# Another comment
https://example.com/feed2.xml
`)); err != nil {
			t.Logf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	// Create syncer with test server URL
	syncer := NewFeedSyncer(repo, "", server.URL)

	// Sync feeds
	ctx := context.Background()
	err := syncer.syncFeeds(ctx)
	if err != nil {
		t.Fatalf("syncFeeds failed: %v", err)
	}

	// Verify only 2 feeds were imported (comments ignored)
	count := testutil.CountFeeds(t, database)
	if count != 2 {
		t.Errorf("Expected 2 feeds (comments skipped), got %d", count)
	}
}

func TestSyncFeeds_PreservesExistingMetadata(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)
	ctx := context.Background()

	// Create an existing feed with custom metadata
	now := time.Now()
	feedID := testutil.CreateTestFeed(t, database, "https://example.com/feed1.xml", false, &now, 5)

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`https://example.com/feed1.xml
https://example.com/feed2.xml`)); err != nil {
			t.Logf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	// Create syncer with test server URL
	syncer := NewFeedSyncer(repo, "", server.URL)

	// Sync feeds
	err := syncer.syncFeeds(ctx)
	if err != nil {
		t.Fatalf("syncFeeds failed: %v", err)
	}

	// Verify 2 feeds exist (1 existing + 1 new)
	count := testutil.CountFeeds(t, database)
	if count != 2 {
		t.Errorf("Expected 2 feeds, got %d", count)
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
}

func TestSyncFeeds_HTTPError(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)

	// Create a test HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create syncer with test server URL
	syncer := NewFeedSyncer(repo, "", server.URL)

	// Sync feeds should fail
	ctx := context.Background()
	err := syncer.syncFeeds(ctx)
	if err == nil {
		t.Fatal("Expected error when HTTP server returns 500, got nil")
	}

	// Verify no feeds were imported
	count := testutil.CountFeeds(t, database)
	if count != 0 {
		t.Errorf("Expected 0 feeds after HTTP error, got %d", count)
	}
}

func TestSyncFeeds_EmptyResponse(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)

	// Create a test HTTP server that returns empty content
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("")); err != nil {
			t.Logf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	// Create syncer with test server URL
	syncer := NewFeedSyncer(repo, "", server.URL)

	// Sync feeds should succeed but import no feeds
	ctx := context.Background()
	err := syncer.syncFeeds(ctx)
	if err != nil {
		t.Fatalf("syncFeeds failed: %v", err)
	}

	// Verify no feeds were imported
	count := testutil.CountFeeds(t, database)
	if count != 0 {
		t.Errorf("Expected 0 feeds from empty response, got %d", count)
	}
}

func TestSyncFeeds_DuplicatesInList(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()
	defer testutil.CleanupTestDB(t, database)

	repo := db.NewFeedRepository(database)

	// Create a test HTTP server with duplicate URLs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`https://example.com/feed1.xml
https://example.com/feed2.xml
https://example.com/feed1.xml
https://example.com/feed3.xml
https://example.com/feed2.xml`)); err != nil {
			t.Logf("error writing response: %v", err)
		}
	}))
	defer server.Close()

	// Create syncer with test server URL
	syncer := NewFeedSyncer(repo, "", server.URL)

	// Sync feeds
	ctx := context.Background()
	err := syncer.syncFeeds(ctx)
	if err != nil {
		t.Fatalf("syncFeeds failed: %v", err)
	}

	// Verify only unique feeds were imported
	count := testutil.CountFeeds(t, database)
	if count != 3 {
		t.Errorf("Expected 3 unique feeds, got %d", count)
	}
}

func TestNewFeedSyncer(t *testing.T) {
	database := testutil.SetupTestDB(t)
	defer database.Close()

	repo := db.NewFeedRepository(database)
	url := "https://example.com/feeds.txt"
	path := "/tmp/feeds.txt"

	syncer := NewFeedSyncer(repo, path, url)

	if syncer == nil {
		t.Fatal("Expected non-nil syncer")
	}

	if syncer.listURL != url {
		t.Errorf("Expected listURL=%s, got %s", url, syncer.listURL)
	}

	if syncer.listPath != path {
		t.Errorf("Expected listPath=%s, got %s", path, syncer.listPath)
	}

	if syncer.repo == nil {
		t.Error("Expected non-nil repo")
	}

	if syncer.httpClient == nil {
		t.Error("Expected non-nil httpClient")
	}
}
