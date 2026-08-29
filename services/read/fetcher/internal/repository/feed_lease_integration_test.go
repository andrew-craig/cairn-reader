//go:build integration
// +build integration

package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestFeedRepository_GetFeedsDueForPolling_CrashRecovery_Integration proves
// the Theme 4 claim on feeds: unlike feed_items/content_outbox, feeds has no
// in-flight status of its own, so a crashed poll needs no X6-style recovery
// -- next_poll_at is simply left in the past and the feed is due again next
// cycle regardless of the lease. What the lease protects here is purely
// exclusivity: while a healthy poll is in flight, a second claimer must not
// pick up the same feed; once the lease expires (simulating a crashed
// poller that never got to record a fetch result), the feed becomes
// claimable again.
func TestFeedRepository_GetFeedsDueForPolling_CrashRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	feedRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feed := &models.Feed{
		FeedURL:     "https://example.com/feed-poll-crash",
		PollingTier: models.PollingTierActive,
		Status:      models.FeedStatusActive,
		NextPollAt:  time.Now().Add(-time.Minute), // already due
	}
	require.NoError(t, feedRepo.Create(ctx, feed))

	claimed, err := feedRepo.GetFeedsDueForPolling(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, feed.ID, claimed[0].ID)

	// Still within lease: a second claimer must not pick up the same feed
	// while the first poll is (notionally) still in flight.
	tooSoon, err := feedRepo.GetFeedsDueForPolling(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, tooSoon, "a fresh lease must not be reselected")

	// Simulate the poller crashing before it ever calls UpdateFetchResult /
	// UpdatePollingInfo: fast-forward the lease into the past.
	_, err = testDB.DB.ExecContext(ctx,
		`UPDATE feeds SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, feed.ID)
	require.NoError(t, err)

	reclaimed, err := feedRepo.GetFeedsDueForPolling(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "expired-lease feed must be reselected")
	require.Equal(t, feed.ID, reclaimed[0].ID)
}

// TestFeedRepository_GetFeedsDueForPolling_Exclusivity_Integration proves
// the atomic claim (Theme 4): concurrent claimers against the same due set
// never claim the same feed twice.
func TestFeedRepository_GetFeedsDueForPolling_Exclusivity_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	feedRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	const seeded = 30
	seededIDs := make(map[uuid.UUID]bool, seeded)
	for i := 0; i < seeded; i++ {
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed-poll-excl-" + uuid.New().String(),
			PollingTier: models.PollingTierActive,
			Status:      models.FeedStatusActive,
			NextPollAt:  time.Now().Add(-time.Minute),
		}
		require.NoError(t, feedRepo.Create(ctx, feed))
		seededIDs[feed.ID] = true
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
			got, err := feedRepo.GetFeedsDueForPolling(ctx, seeded)
			require.NoError(t, err)
			mu.Lock()
			for _, f := range got {
				claimedByAll[f.ID]++
			}
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	require.Len(t, claimedByAll, seeded, "every seeded feed must be claimed exactly once, none lost")
	for id, count := range claimedByAll {
		require.True(t, seededIDs[id], "claimed unexpected feed %s", id)
		require.Equal(t, 1, count, "feed %s claimed more than once", id)
	}
}
