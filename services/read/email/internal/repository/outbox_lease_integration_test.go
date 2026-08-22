//go:build integration
// +build integration

package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// seedRawEmailForOutbox inserts a minimal raw_emails row so content_outbox's
// FK constraint is satisfied.
func seedRawEmailForOutbox(t *testing.T, ctx context.Context, rawRepo RawEmailRepository) uuid.UUID {
	t.Helper()
	email := newTestRawEmail()
	email.ProcessingStatus = models.ProcessingStatusCompleted
	require.NoError(t, rawRepo.Create(ctx, email))
	return email.ID
}

func newTestEmailOutboxEntry(rawEmailID uuid.UUID) *models.ContentOutbox {
	return &models.ContentOutbox{
		RawEmailID:     rawEmailID,
		ContentPayload: map[string]interface{}{"title": "Test"},
		UserID:         uuid.New(),
		DeliveryStatus: models.DeliveryStatusPending,
		MaxRetries:     6,
		NextRetryAt:    time.Now(),
	}
}

// TestOutboxRepository_GetPendingEntries_CrashRecovery_Integration proves
// Theme 4's atomic claim on the email outbox: this table's X6 selector
// (delivery_status IN ('pending','sending')) was already correct, but
// without an atomic claim a widened selector risks double delivery. The
// lease closes that gap: a claimed-but-crashed entry is only reselected
// once its lease expires, and a still-healthy in-flight delivery is never
// preempted.
func TestOutboxRepository_GetPendingEntries_CrashRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	rawRepo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	rawEmailID := seedRawEmailForOutbox(t, ctx, rawRepo)
	entry := newTestEmailOutboxEntry(rawEmailID)
	require.NoError(t, outboxRepo.Create(ctx, entry))

	claimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, models.DeliveryStatusSending, claimed[0].DeliveryStatus)

	tooSoon, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, tooSoon, "a fresh lease must not be reselected")

	_, err = testDB.DB.ExecContext(ctx,
		`UPDATE content_outbox SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, entry.ID)
	require.NoError(t, err)

	reclaimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "expired-lease entry must be reselected")
	require.Equal(t, entry.ID, reclaimed[0].ID)
}

// TestOutboxRepository_GetPendingEntries_NULLLease_Reclaimed_Integration
// reproduces the state every pre-existing 'sending' row is in immediately
// after the lease_expires_at column is added: NULL. `NULL < now()` evaluates
// to NULL/false in SQL, so a naive `delivery_status = 'sending' AND
// lease_expires_at < now()` predicate would never re-select these rows --
// silently leaving the entire existing X6 backlog stranded even after this
// fix ships. The selector must treat a NULL lease the same as an expired one.
func TestOutboxRepository_GetPendingEntries_NULLLease_Reclaimed_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	rawRepo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	rawEmailID := seedRawEmailForOutbox(t, ctx, rawRepo)
	entry := newTestEmailOutboxEntry(rawEmailID)
	require.NoError(t, outboxRepo.Create(ctx, entry))

	// Simulate an entry already stranded in 'sending' from before this
	// migration -- lease_expires_at defaults to NULL, never set by anything.
	_, err := testDB.DB.ExecContext(ctx,
		`UPDATE content_outbox SET delivery_status = 'sending', lease_expires_at = NULL WHERE id = $1`, entry.ID)
	require.NoError(t, err)

	reclaimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "a pre-existing NULL-lease 'sending' entry must be reclaimable, not permanently stranded")
	require.Equal(t, entry.ID, reclaimed[0].ID)
}

// TestOutboxRepository_UpdateRetryInfo_ImmediatelyReselectable_Integration
// proves the email outbox retry ladder is reachable: a failed-but-retryable
// delivery attempt must be reselectable on the very next poll instead of
// waiting out the full claim lease.
func TestOutboxRepository_UpdateRetryInfo_ImmediatelyReselectable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	outboxRepo := NewOutboxRepository(testDB.DB)
	rawRepo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	rawEmailID := seedRawEmailForOutbox(t, ctx, rawRepo)
	entry := newTestEmailOutboxEntry(rawEmailID)
	require.NoError(t, outboxRepo.Create(ctx, entry))

	claimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	// nextRetryAt is deliberately "now" so this test isolates the
	// lease-reset behavior from next_retry_at backoff scheduling.
	require.NoError(t, outboxRepo.UpdateRetryInfo(ctx, entry.ID, 1, time.Now(), "delivery failed"))

	reclaimed, err := outboxRepo.GetPendingEntries(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "retried entry must be immediately reselectable")
	require.Equal(t, entry.ID, reclaimed[0].ID)
	require.Equal(t, 1, reclaimed[0].RetryCount)
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
	rawRepo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	rawEmailID := seedRawEmailForOutbox(t, ctx, rawRepo)

	const seeded = 30
	seededIDs := make(map[uuid.UUID]bool, seeded)
	for i := 0; i < seeded; i++ {
		entry := newTestEmailOutboxEntry(rawEmailID)
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
