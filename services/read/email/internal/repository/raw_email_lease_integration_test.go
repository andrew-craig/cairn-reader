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

func newTestRawEmail() *models.RawEmail {
	html := "<html><body>hi</body></html>"
	return &models.RawEmail{
		UserID:           uuid.New(),
		Recipient:        "user@read.cairnapp.com",
		SenderEmail:      "sender@example.com",
		HTMLBody:         &html,
		ReceivedAt:       time.Now(),
		ProcessingStatus: models.ProcessingStatusPending,
	}
}

// TestRawEmailRepository_GetPendingEmails_CrashRecovery_Integration
// reproduces audit X6: a worker claims an email into 'processing' and then
// crashes before ever calling UpdateStatus or UpdateError again. On main
// (WHERE processing_status = $1 called only with 'pending'), no query ever
// re-selects the email -- both retry caps become unreachable.
func TestRawEmailRepository_GetPendingEmails_CrashRecovery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	email := newTestRawEmail()
	require.NoError(t, repo.Create(ctx, email))

	claimed, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, models.ProcessingStatusProcessing, claimed[0].ProcessingStatus)

	tooSoon, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, tooSoon, "a fresh lease must not be reselected")

	_, err = testDB.DB.ExecContext(ctx,
		`UPDATE raw_emails SET lease_expires_at = now() - interval '1 minute' WHERE id = $1`, email.ID)
	require.NoError(t, err)

	reclaimed, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "expired-lease email must be reselected")
	require.Equal(t, email.ID, reclaimed[0].ID)
}

// TestRawEmailRepository_UpdateError_ImmediatelyReselectable_Integration
// proves the raw-email retry ladder is reachable: a failed-but-retryable
// email must be reselectable on the very next poll instead of waiting out
// the full claim lease.
func TestRawEmailRepository_UpdateError_ImmediatelyReselectable_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	email := newTestRawEmail()
	require.NoError(t, repo.Create(ctx, email))

	claimed, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, repo.UpdateError(ctx, email.ID, 1, "clean failed"))

	reclaimed, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Len(t, reclaimed, 1, "retried email must be immediately reselectable")
	require.Equal(t, email.ID, reclaimed[0].ID)
	require.Equal(t, 1, reclaimed[0].RetryCount)
}

// TestRawEmailRepository_UpdateError_TerminalFailure_NotReselected_Integration
// confirms that once retry_count reaches 5, UpdateError moves the email to
// the terminal 'failed' status and it drops out of the widened selector
// entirely (retry_count < 5 guard).
func TestRawEmailRepository_UpdateError_TerminalFailure_NotReselected_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	email := newTestRawEmail()
	require.NoError(t, repo.Create(ctx, email))

	claimed, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, repo.UpdateError(ctx, email.ID, 5, "clean failed"))

	reclaimed, err := repo.GetPendingEmails(ctx, 10)
	require.NoError(t, err)
	require.Empty(t, reclaimed, "terminally-failed email must not be reselected")

	var status models.ProcessingStatus
	require.NoError(t, testDB.DB.QueryRowContext(ctx,
		`SELECT processing_status FROM raw_emails WHERE id = $1`, email.ID).Scan(&status))
	require.Equal(t, models.ProcessingStatusFailed, status)
}

// TestRawEmailRepository_GetPendingEmails_Exclusivity_Integration proves the
// atomic claim (Theme 4): concurrent claimers against the same pending set
// never claim the same row twice.
func TestRawEmailRepository_GetPendingEmails_Exclusivity_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewRawEmailRepository(testDB.DB)
	ctx := context.Background()

	const seeded = 30
	seededIDs := make(map[uuid.UUID]bool, seeded)
	for i := 0; i < seeded; i++ {
		email := newTestRawEmail()
		require.NoError(t, repo.Create(ctx, email))
		seededIDs[email.ID] = true
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
			got, err := repo.GetPendingEmails(ctx, seeded)
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
