package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-explore/fetcher/internal/testutil"
)

func TestGetNextFeed_PrioritizesNeverFetched(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create feeds: one never fetched, two with different last fetch times
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	weekAgo := now.Add(-7 * 24 * time.Hour)

	neverFetchedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL2, true, &yesterday, 0)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL3, true, &weekAgo, 0)

	// GetNextFeed should return the never-fetched feed
	feed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed == nil {
		t.Fatal("Expected feed, got nil")
	}

	if feed.ID != neverFetchedID {
		t.Errorf("Expected never-fetched feed (ID %d), got ID %d", neverFetchedID, feed.ID)
	}

	if feed.LastFetchedAt != nil {
		t.Error("Expected LastFetchedAt to be nil for never-fetched feed")
	}
}

func TestGetNextFeed_PrioritizesOldestFetch(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create feeds with different last fetch times (no never-fetched feeds)
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	weekAgo := now.Add(-7 * 24 * time.Hour)

	testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, &yesterday, 0)
	oldestID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL2, true, &weekAgo, 0)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL3, true, &now, 0)

	// GetNextFeed should return the oldest-fetched feed
	feed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed == nil {
		t.Fatal("Expected feed, got nil")
	}

	if feed.ID != oldestID {
		t.Errorf("Expected oldest feed (ID %d), got ID %d", oldestID, feed.ID)
	}
}

func TestGetNextFeed_SkipsDisabledFeeds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create one disabled never-fetched feed and one enabled feed
	now := time.Now()
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, false, nil, 10) // Disabled
	enabledID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL2, true, &now, 0)

	// GetNextFeed should skip the disabled feed
	feed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed == nil {
		t.Fatal("Expected feed, got nil")
	}

	if feed.ID != enabledID {
		t.Errorf("Expected enabled feed (ID %d), got ID %d", enabledID, feed.ID)
	}
}

func TestGetNextFeed_NoEnabledFeeds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create only disabled feeds
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, false, nil, 10)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL2, false, nil, 10)

	// GetNextFeed should return nil
	feed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if feed != nil {
		t.Error("Expected nil feed when no enabled feeds exist")
	}
}

func TestUpdateFetchResult_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create a feed with failures
	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 5)

	// Update with success
	err := repo.UpdateFetchResult(ctx, feedID, true)
	if err != nil {
		t.Fatalf("UpdateFetchResult failed: %v", err)
	}

	// Verify last_fetched_at is set and consecutive_failures is reset
	_, _, failures, lastFetched := testutil.GetFeedByID(t, db, feedID)

	if failures != 0 {
		t.Errorf("Expected consecutive_failures=0, got %d", failures)
	}

	if lastFetched == nil {
		t.Error("Expected last_fetched_at to be set")
	} else if time.Since(*lastFetched) > 5*time.Second {
		t.Error("Expected last_fetched_at to be recent")
	}
}

func TestUpdateFetchResult_Failure(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create a feed with no failures
	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)

	// Update with failure
	err := repo.UpdateFetchResult(ctx, feedID, false)
	if err != nil {
		t.Fatalf("UpdateFetchResult failed: %v", err)
	}

	// Verify consecutive_failures is incremented
	_, enabled, failures, _ := testutil.GetFeedByID(t, db, feedID)

	if failures != 1 {
		t.Errorf("Expected consecutive_failures=1, got %d", failures)
	}

	if !enabled {
		t.Error("Expected feed to remain enabled after 1 failure")
	}
}

func TestUpdateFetchResult_DisablesAfter10Failures(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create a feed with 9 failures
	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 9)

	// Update with failure (should disable)
	err := repo.UpdateFetchResult(ctx, feedID, false)
	if err != nil {
		t.Fatalf("UpdateFetchResult failed: %v", err)
	}

	// Verify feed is disabled
	_, enabled, failures, _ := testutil.GetFeedByID(t, db, feedID)

	if failures != 10 {
		t.Errorf("Expected consecutive_failures=10, got %d", failures)
	}

	if enabled {
		t.Error("Expected feed to be disabled after 10 failures")
	}
}

func TestImportFeeds_NewFeeds(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Import new feeds
	feedURLs := []string{
		testutil.TestFeedURL1,
		testutil.TestFeedURL2,
		testutil.TestFeedURL3,
	}

	err := repo.ImportFeeds(ctx, feedURLs)
	if err != nil {
		t.Fatalf("ImportFeeds failed: %v", err)
	}

	// Verify feeds were created
	count := testutil.CountFeeds(t, db)
	if count != 3 {
		t.Errorf("Expected 3 feeds, got %d", count)
	}
}

func TestImportFeeds_DuplicatesIgnored(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create a feed
	now := time.Now()
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, false, &now, 5)

	// Import feeds including the duplicate
	feedURLs := []string{
		testutil.TestFeedURL1, // Duplicate
		testutil.TestFeedURL2,
		testutil.TestFeedURL3,
	}

	err := repo.ImportFeeds(ctx, feedURLs)
	if err != nil {
		t.Fatalf("ImportFeeds failed: %v", err)
	}

	// Verify only 3 feeds exist (duplicate was ignored)
	count := testutil.CountFeeds(t, db)
	if count != 3 {
		t.Errorf("Expected 3 feeds, got %d", count)
	}

	// Verify existing feed metadata was preserved
	_, enabled, failures, lastFetched := testutil.GetFeedByID(t, db, 1)
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

func TestRecordFetchHistory_Success(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create a feed
	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)

	// Record successful fetch
	err := repo.RecordFetchHistory(ctx, feedID, true, 10, 8, "")
	if err != nil {
		t.Fatalf("RecordFetchHistory failed: %v", err)
	}

	// Verify history record was created
	count := testutil.CountFetchHistory(t, db)
	if count != 1 {
		t.Errorf("Expected 1 fetch history record, got %d", count)
	}

	// Verify record contents
	var success bool
	var articlesFound, articlesSent int
	var errorMsg sql.NullString
	err = db.QueryRow(`
		SELECT success, articles_found, articles_sent, error_message
		FROM fetch_history
		WHERE feed_id = $1
	`, feedID).Scan(&success, &articlesFound, &articlesSent, &errorMsg)

	if err != nil {
		t.Fatalf("Failed to query fetch history: %v", err)
	}

	if !success {
		t.Error("Expected success=true")
	}
	if articlesFound != 10 {
		t.Errorf("Expected articles_found=10, got %d", articlesFound)
	}
	if articlesSent != 8 {
		t.Errorf("Expected articles_sent=8, got %d", articlesSent)
	}
	if errorMsg.Valid {
		t.Error("Expected no error message for successful fetch")
	}
}

func TestRecordFetchHistory_Failure(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create a feed
	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)

	// Record failed fetch
	err := repo.RecordFetchHistory(ctx, feedID, false, 0, 0, "HTTP timeout")
	if err != nil {
		t.Fatalf("RecordFetchHistory failed: %v", err)
	}

	// Verify history record was created
	count := testutil.CountFetchHistory(t, db)
	if count != 1 {
		t.Errorf("Expected 1 fetch history record, got %d", count)
	}

	// Verify record contents
	var success bool
	var errorMsg sql.NullString
	err = db.QueryRow(`
		SELECT success, error_message
		FROM fetch_history
		WHERE feed_id = $1
	`, feedID).Scan(&success, &errorMsg)

	if err != nil {
		t.Fatalf("Failed to query fetch history: %v", err)
	}

	if success {
		t.Error("Expected success=false")
	}
	if !errorMsg.Valid || errorMsg.String != "HTTP timeout" {
		t.Errorf("Expected error_message='HTTP timeout', got %v", errorMsg)
	}
}

func TestListFeeds_All(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create feeds (some enabled, some disabled)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL2, false, nil, 10)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL3, true, nil, 0)

	// List all feeds
	feeds, err := repo.ListFeeds(ctx, false)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}

	if len(feeds) != 3 {
		t.Errorf("Expected 3 feeds, got %d", len(feeds))
	}
}

func TestListFeeds_EnabledOnly(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	// Create feeds (some enabled, some disabled)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL2, false, nil, 10)
	testutil.CreateTestFeed(t, db, testutil.TestFeedURL3, true, nil, 0)

	// List enabled feeds only
	feeds, err := repo.ListFeeds(ctx, true)
	if err != nil {
		t.Fatalf("ListFeeds failed: %v", err)
	}

	if len(feeds) != 2 {
		t.Errorf("Expected 2 enabled feeds, got %d", len(feeds))
	}

	// Verify all returned feeds are enabled
	for _, feed := range feeds {
		if !feed.Enabled {
			t.Errorf("Expected only enabled feeds, got disabled feed: %s", feed.URL)
		}
	}
}
