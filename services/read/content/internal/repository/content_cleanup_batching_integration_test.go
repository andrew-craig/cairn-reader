//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestContentRepository_DeleteOrphaned_Batching_Integration reproduces the
// "Cleanup batching" finding: DeleteOrphaned had no LIMIT, so a single call
// deleted every orphaned row in one long-lock-holding transaction regardless
// of the batch size the caller intended. Against real Postgres, a single
// call must delete at most batchSize rows, and repeated calls must be
// required to clear a backlog larger than one batch.
func TestContentRepository_DeleteOrphaned_Batching_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	const (
		totalOrphaned = 5
		batchSize     = 2
	)

	orphanedAt := time.Now().Add(-100 * 24 * time.Hour) // older than the 90-day cutoff
	for i := 0; i < totalOrphaned; i++ {
		_, err := testDB.DB.ExecContext(ctx, `
			INSERT INTO contents (id, content_hash, cleaned_html, original_url, title, source_type, orphaned_at)
			VALUES ($1, $2, $3, $4, $5, 'web', $6)
		`, uuid.New(), fmt.Sprintf("hash-%d", i), "<p>orphaned</p>", fmt.Sprintf("https://example.com/%d", i), fmt.Sprintf("Orphaned %d", i), orphanedAt)
		require.NoError(t, err)
	}

	olderThan := 90 * 24 * time.Hour

	// First call must be bounded to batchSize, not delete the whole backlog.
	deleted, err := repo.DeleteOrphaned(ctx, olderThan, batchSize)
	require.NoError(t, err)
	require.EqualValues(t, batchSize, deleted, "a single call must delete at most batchSize rows")

	var remaining int
	err = testDB.DB.QueryRowContext(ctx, `SELECT count(*) FROM contents WHERE orphaned_at IS NOT NULL`).Scan(&remaining)
	require.NoError(t, err)
	require.Equal(t, totalOrphaned-batchSize, remaining, "rows beyond the batch must survive the first pass")

	// Drain the rest in bounded passes, mirroring how CleanupJob.Run loops.
	totalDeleted := int64(batchSize)
	passes := 1
	for {
		n, err := repo.DeleteOrphaned(ctx, olderThan, batchSize)
		require.NoError(t, err)
		if n == 0 {
			break
		}
		require.LessOrEqual(t, n, int64(batchSize), "every pass must stay within batchSize")
		totalDeleted += n
		passes++
		require.Less(t, passes, totalOrphaned+2, "must not loop indefinitely")
	}

	require.EqualValues(t, totalOrphaned, totalDeleted, "repeated bounded passes must eventually clear the full backlog")
	require.Greater(t, passes, 1, "clearing more than one batch must take more than one pass")

	err = testDB.DB.QueryRowContext(ctx, `SELECT count(*) FROM contents WHERE orphaned_at IS NOT NULL`).Scan(&remaining)
	require.NoError(t, err)
	require.Equal(t, 0, remaining)
}
