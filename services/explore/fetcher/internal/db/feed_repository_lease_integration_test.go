//go:build integration

package db

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/explore/fetcher/internal/testutil"
)

// TestGetNextFeed_CrashRecovery_Integration proves the Theme 4 claim on
// feeds: this table has no in-flight status of its own (see
// GetNextFeed's doc comment), so a crashed fetch needs no X6-style recovery
// -- last_fetched_at is simply left untouched and the feed is picked first
// again next cycle regardless of the lease. What the lease protects here is
// purely exclusivity: while a fetch is (notionally) in flight, a second
// claimer must not pick up the same feed; once the lease expires (simulating
// a fetcher that crashed before ever calling UpdateFetchResult), the feed
// becomes claimable again.
func TestGetNextFeed_CrashRecovery_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)

	claimed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}
	if claimed == nil || claimed.ID != feedID {
		t.Fatalf("expected feed %d to be claimed, got %+v", feedID, claimed)
	}

	// Still within lease: a second claimer must not pick up the same feed
	// while the first fetch is (notionally) still in flight.
	tooSoon, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}
	if tooSoon != nil {
		t.Fatalf("expected no feed to be claimable under a fresh lease, got %+v", tooSoon)
	}

	// Simulate the fetcher crashing before it ever calls UpdateFetchResult:
	// fast-forward the lease into the past.
	if _, err := db.Exec(ctx,
		`UPDATE feeds SET fetch_lease_expires_at = now() - interval '1 minute' WHERE id = $1`, feedID); err != nil {
		t.Fatalf("failed to expire lease: %v", err)
	}

	reclaimed, err := repo.GetNextFeed(ctx)
	if err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}
	if reclaimed == nil || reclaimed.ID != feedID {
		t.Fatalf("expected expired-lease feed %d to be reselected, got %+v", feedID, reclaimed)
	}
}

// TestUpdateFetchResult_ClearsLease_Integration confirms that a completed
// fetch attempt (success or failure) releases the lease immediately rather
// than waiting it out, so GetNextFeed's normal least-recently-fetched
// ordering resumes controlling feed selection right away.
func TestUpdateFetchResult_ClearsLease_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := testutil.CreateTestFeed(t, db, testutil.TestFeedURL1, true, nil, 0)

	if _, err := repo.GetNextFeed(ctx); err != nil {
		t.Fatalf("GetNextFeed failed: %v", err)
	}

	if err := repo.UpdateFetchResult(ctx, feedID, true, "", ""); err != nil {
		t.Fatalf("UpdateFetchResult failed: %v", err)
	}

	var leaseExpiresAt *time.Time
	if err := db.QueryRow(ctx, `SELECT fetch_lease_expires_at FROM feeds WHERE id = $1`, feedID).Scan(&leaseExpiresAt); err != nil {
		t.Fatalf("failed to read lease: %v", err)
	}
	if leaseExpiresAt != nil {
		t.Fatalf("expected lease to be cleared after UpdateFetchResult, got %v", leaseExpiresAt)
	}
}

// TestGetNextFeed_Exclusivity_Integration proves the atomic claim (Theme 4):
// concurrent claimers never claim the same feed twice.
func TestGetNextFeed_Exclusivity_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	defer testutil.CleanupTestDB(t, db)

	repo := NewFeedRepository(db)
	ctx := context.Background()

	const seeded = 20
	seededIDs := make(map[int]bool, seeded)
	for i := 0; i < seeded; i++ {
		id := testutil.CreateTestFeed(t, db, fmt.Sprintf("https://example.com/excl-feed-%d.xml", i), true, nil, 0)
		seededIDs[id] = true
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	claimedByAll := make(map[int]int)
	start := make(chan struct{})

	for i := 0; i < seeded; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := repo.GetNextFeed(ctx)
			if err != nil {
				t.Errorf("GetNextFeed failed: %v", err)
				return
			}
			if got == nil {
				return
			}
			mu.Lock()
			claimedByAll[got.ID]++
			mu.Unlock()
		}()
	}
	close(start)
	wg.Wait()

	if len(claimedByAll) != seeded {
		t.Fatalf("expected all %d seeded feeds to be claimed exactly once, got %d distinct claims", seeded, len(claimedByAll))
	}
	for id, count := range claimedByAll {
		if !seededIDs[id] {
			t.Errorf("claimed unexpected feed %d", id)
		}
		if count != 1 {
			t.Errorf("feed %d claimed %d times, want 1", id, count)
		}
	}
}
