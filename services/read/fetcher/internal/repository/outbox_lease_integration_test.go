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

// seedFeedItemForOutbox inserts a minimal feed + feed_item pair so
// content_outbox's FK constraint is satisfied.
func seedFeedItemForOutbox(t *testing.T, ctx context.Context, feedRepo FeedRepository, itemRepo FeedItemRepository, urlSuffix string) uuid.UUID {
	t.Helper()
	feed := &models.Feed{
		FeedURL:     "https://example.com/feed-outbox-" + urlSuffix,
		PollingTier: models.PollingTierActive,
		Status:      models.FeedStatusActive,
		NextPollAt:  time.Now(),
	}
	require.NoError(t, feedRepo.Create(ctx, feed))

	item := &models.FeedItem{
		FeedID:           feed.ID,
		ItemURL:          "https://example.com/article-outbox-" + urlSuffix,
		ProcessingStatus: models.ProcessingStatusCompleted,
	}
	require.NoError(t, itemRepo.Create(ctx, item))
	return item.ID
}

func newTestOutboxEntry(feedItemID uuid.UUID) *models.ContentOutbox {
	return &models.ContentOutbox{
		FeedItemID:     feedItemID,
		ContentPayload: map[string]interface{}{"title": "Test"},
		UserIDs:        []uuid.UUID{uuid.New()},
		DeliveryStatus: models.DeliveryStatusPending,
		MaxRetries:     6,
		NextRetryAt:    time.Now(),
	}
}

// TestOutboxRepository_GetPendingEntries_CrashRecovery_Integration
// reproduces audit X6's most severe case: an entry is claimed into 'sending'
// and the worker crashes mid-delivery before ever calling
// UpdateDeliveryStatus or IncrementRetryCount again. On main (WHERE
// delivery_status = 'pending' only), the entry is stranded forever -- and
// because there was never an atomic claim, a naive fix of widening the
// selector to include 'sending' would double-deliver a still-in-flight
// entry. The lease is what makes "stranded" and "double-delivered" both
// impossible at once.
func TestOutboxRepository_GetPendingEntries_CrashRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	itemRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedItemID := seedFeedItemForOutbox(t, ctx, fRepo, itemRepo, "crash")
	entry := newTestOutboxEntry(feedItemID)
	require.NoError(t, outboxRepo.Create(ctx, entry))

	claimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, models.DeliveryStatusSending, claimed[0].DeliveryStatus)

	// Still within lease: a healthy in-progress delivery must not be
	// re-claimed by another worker (that would double-deliver).
	tooSoon, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, tooSoon, "a fresh lease must not be reselected")

	// Simulate lease expiry after a crash.
	_, err = testDB.DB.ExecContext(ctx,
		`UPDATE content_outbox SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, entry.ID)
	require.NoError(t, err)

	reclaimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "expired-lease entry must be reselected")
	require.Equal(t, entry.ID, reclaimed[0].ID)
	require.Equal(t, models.DeliveryStatusSending, reclaimed[0].DeliveryStatus)
}

// TestOutboxRepository_IncrementRetryCount_ImmediatelyReselectable_Integration
// proves the outbox retry ladder is reachable: a failed-but-retryable
// delivery attempt must be reselectable on the very next poll instead of
// waiting out the full claim lease.
func TestOutboxRepository_IncrementRetryCount_ImmediatelyReselectable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	itemRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedItemID := seedFeedItemForOutbox(t, ctx, fRepo, itemRepo, "retry")
	entry := newTestOutboxEntry(feedItemID)
	require.NoError(t, outboxRepo.Create(ctx, entry))

	claimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	// nextRetryAt is deliberately "now" (not a future backoff time) so this
	// test isolates the lease-reset behavior from next_retry_at scheduling --
	// the widened selector also requires next_retry_at <= now(), which is a
	// separate, orthogonal gate from the lease.
	require.NoError(t, outboxRepo.IncrementRetryCount(ctx, entry.ID, time.Now(), "delivery failed"))

	reclaimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "retried entry must be immediately reselectable")
	require.Equal(t, entry.ID, reclaimed[0].ID)
	require.Equal(t, 1, reclaimed[0].RetryCount)
}

// TestOutboxRepository_IncrementRetryCount_TerminalFailure_NotReselected_Integration
// confirms that once retry_count reaches max_retries, IncrementRetryCount
// moves the entry to the terminal 'failed' status and it drops out of the
// widened selector entirely (matching the reference email-outbox pattern's
// retry_count < max_retries guard).
func TestOutboxRepository_IncrementRetryCount_TerminalFailure_NotReselected_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	itemRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedItemID := seedFeedItemForOutbox(t, ctx, fRepo, itemRepo, "terminal")
	entry := newTestOutboxEntry(feedItemID)
	entry.MaxRetries = 1
	require.NoError(t, outboxRepo.Create(ctx, entry))

	claimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, outboxRepo.IncrementRetryCount(ctx, entry.ID, time.Now().Add(time.Hour), "delivery failed"))

	reclaimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, reclaimed, "terminally-failed entry must not be reselected")

	var status models.DeliveryStatus
	require.NoError(t, testDB.DB.QueryRowContext(ctx,
		`SELECT delivery_status FROM content_outbox WHERE id = $1`, entry.ID).Scan(&status))
	require.Equal(t, models.DeliveryStatusFailed, status)
}

// TestOutboxRepository_GetPendingEntries_Exclusivity_Integration proves the
// atomic claim (Theme 4): concurrent claimers against the same pending set
// never claim the same row twice.
func TestOutboxRepository_GetPendingEntries_Exclusivity_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	itemRepo := NewFeedItemRepository(testDB.DB)
	fRepo := NewFeedRepository(testDB.DB)
	ctx := context.Background()

	feedItemID := seedFeedItemForOutbox(t, ctx, fRepo, itemRepo, "exclusivity")

	const seeded = 30
	seededIDs := make(map[uuid.UUID]bool, seeded)
	for i := 0; i < seeded; i++ {
		entry := newTestOutboxEntry(feedItemID)
		require.NoError(t, outboxRepo.Create(ctx, entry))
		seededIDs[entry.ID] = true
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
			got, err := outboxRepo.GetPendingEntries(ctx, seeded)
			require.NoError(t, err)
			mu.Lock()
			for _, e := range got {
				claimedByAll[e.ID]++
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
