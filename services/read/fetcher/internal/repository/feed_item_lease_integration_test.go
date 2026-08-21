//go:build integration
// +build integration

package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// seedFeedForItems inserts a minimal feed row so feed_items' FK constraint is
// satisfied.
func seedFeedForItems(t *testing.T, ctx context.Context, feedRepo FeedRepository, urlSuffix string) uuid.UUID {
	t.Helper()
	feed := &models.Feed{
		FeedURL:     "https://example.com/feed-" + urlSuffix,
		PollingTier: models.PollingTierActive,
		Status:      models.FeedStatusActive,
		NextPollAt:  time.Now(),
	}
	require.NoError(t, feedRepo.Create(ctx, feed))
	return feed.ID
}

// TestFeedItemRepository_GetPendingItems_CrashRecovery_Integration reproduces
// audit X6: a worker claims an item into 'processing' and then crashes before
// ever calling IncrementRetryCount or UpdateProcessingStatus again. On main
// (WHERE processing_status = 'pending' only), no query ever re-selects the
// item -- it is stranded forever. With the claim-lease pattern, the item
// becomes reselectable once its lease expires.
func TestFeedItemRepository_GetPendingItems_CrashRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	feedRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedID := seedFeedForItems(t, ctx, fRepo, "crash")
	item := &models.FeedItem{
		FeedID:           feedID,
		ItemURL:          "https://example.com/article-crash",
		ProcessingStatus: models.ProcessingStatusPending,
	}
	require.NoError(t, feedRepo.Create(ctx, item))

	// First claim: the item moves to 'processing' with a fresh 10-minute lease.
	claimed, err := feedRepo.GetPendingItems(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, item.ID, claimed[0].ID)
	require.Equal(t, models.ProcessingStatusProcessing, claimed[0].ProcessingStatus)

	// The "worker" now crashes: no IncrementRetryCount, no
	// UpdateProcessingStatus. A poll immediately after must NOT re-select the
	// item -- its lease is still valid, so a still-healthy worker should not
	// be preempted by another claimer.
	reclaimedTooSoon, err := feedRepo.GetPendingItems(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, reclaimedTooSoon, "a fresh lease must not be reselected")

	// Simulate the lease expiring (a real crash would just wait out the 10
	// minutes; we fast-forward by writing the lease into the past directly).
	_, err = testDB.DB.ExecContext(ctx,
		`UPDATE feed_items SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, item.ID)
	require.NoError(t, err)

	// The next poll must re-select the stranded item -- this is the assertion
	// that fails on main, where GetPendingItems only ever looks at 'pending'.
	reclaimed, err := feedRepo.GetPendingItems(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "expired-lease item must be reselected")
	require.Equal(t, item.ID, reclaimed[0].ID)
	require.Equal(t, models.ProcessingStatusProcessing, reclaimed[0].ProcessingStatus)
}

// TestFeedItemRepository_IncrementRetryCount_ImmediatelyReselectable_Integration
// proves the retry ladder is actually reachable: after a failed processing
// attempt that will be retried (not yet at max retries), the item must be
// reselectable on the very next poll rather than waiting out the full lease.
func TestFeedItemRepository_IncrementRetryCount_ImmediatelyReselectable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	feedRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedID := seedFeedForItems(t, ctx, fRepo, "retry")
	item := &models.FeedItem{
		FeedID:           feedID,
		ItemURL:          "https://example.com/article-retry",
		ProcessingStatus: models.ProcessingStatusPending,
	}
	require.NoError(t, feedRepo.Create(ctx, item))

	claimed, err := feedRepo.GetPendingItems(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, feedRepo.IncrementRetryCount(ctx, item.ID, "fetch failed"))

	// No lease-expiry wait: the retry-increment call itself must reset the
	// lease so the item is reselectable on the very next poll.
	reclaimed, err := feedRepo.GetPendingItems(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "retried item must be immediately reselectable")
	require.Equal(t, item.ID, reclaimed[0].ID)
	require.Equal(t, 1, reclaimed[0].RetryCount)
}

// TestFeedItemRepository_GetPendingItems_Exclusivity_Integration proves the
// atomic claim (Theme 4): concurrent claimers against the same pending set
// never claim the same row twice.
func TestFeedItemRepository_GetPendingItems_Exclusivity_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	feedRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedID := seedFeedForItems(t, ctx, fRepo, "exclusivity")

	const seeded = 30
	seededIDs := make(map[uuid.UUID]bool, seeded)
	for i := 0; i < seeded; i++ {
		item := &models.FeedItem{
			FeedID:           feedID,
			ItemURL:          "https://example.com/article-excl",
			ProcessingStatus: models.ProcessingStatusPending,
		}
		require.NoError(t, feedRepo.Create(ctx, item))
		seededIDs[item.ID] = true
	}

	const claimers = 6
	var wg sync.WaitGroup
	var mu sync.Mutex
	claimedByAll := make(map[uuid.UUID]int)
	start := make(chan struct{})

	for i := 0; i < claimers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			// Each claimer requests a batch overlapping the full seeded set,
			// so without FOR UPDATE SKIP LOCKED they would race for the same rows.
			got, err := feedRepo.GetPendingItems(ctx, seeded)
			require.NoError(t, err)
			mu.Lock()
			for _, it := range got {
				claimedByAll[it.ID]++
			}
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	require.Len(t, claimedByAll, seeded, "every seeded row must be claimed exactly once, none lost")
	for id, count := range claimedByAll {
		require.True(t, seededIDs[id], "claimed unexpected row %s", id)
		require.Equal(t, 1, count, "row %s claimed more than once", id)
	}
}
