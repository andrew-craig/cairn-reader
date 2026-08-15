//go:build integration
// +build integration

package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestContentRepository_ConcurrentCreate_RSS_Integration reproduces the
// dedup-index finding: idx_contents_rss_dedup used to be a plain (non-unique)
// index, so two concurrent deliveries of the same RSS item both pass any
// application-level "not found" check and both insert, producing duplicate
// content rows. Against real Postgres with the UNIQUE dedup index + ON
// CONFLICT DO UPDATE in Create, concurrent inserts of identical content must
// converge on exactly one row.
func TestContentRepository_ConcurrentCreate_RSS_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()
	feedID := uuid.New()

	const concurrency = 10
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			content := &models.Content{
				ContentHash:  "concurrent-hash",
				CleanedHTML:  "<p>same content</p>",
				OriginalURL:  "https://example.com/concurrent-article",
				Title:        "Concurrent Article",
				SourceType:   models.SourceTypeRSS,
				SourceFeedID: &feedID,
			}
			errs[i] = repo.Create(ctx, content)
			ids[i] = content.ID
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "concurrent create %d must not error", i)
	}

	first := ids[0]
	for i, id := range ids {
		require.Equal(t, first, id, "concurrent create %d must resolve to the same persisted row", i)
	}

	var count int
	err := testDB.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM contents WHERE content_hash = $1 AND source_feed_id = $2`,
		"concurrent-hash", feedID,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "concurrent inserts of identical RSS content must produce exactly one row")
}

// TestContentRepository_ConcurrentBulkCreate_Email_Integration is the
// non-RSS analog: email/manual content has no feed to dedupe on, so it
// relies on the new (content_hash, original_url) unique index instead.
// Concurrent bulk deliveries of the same email content (e.g. two outbox
// workers racing on a retry) must converge on exactly one row.
func TestContentRepository_ConcurrentBulkCreate_Email_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()
	emailURL := "email://" + uuid.New().String()

	const concurrency = 10
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			content := &models.Content{
				ContentHash: "concurrent-email-hash",
				CleanedHTML: "<p>same email content</p>",
				OriginalURL: emailURL,
				Title:       "Concurrent Email",
				SourceType:  models.SourceTypeEmail,
			}
			errs[i] = repo.BulkCreate(ctx, []*models.Content{content})
			ids[i] = content.ID
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "concurrent bulk create %d must not error", i)
	}

	first := ids[0]
	for i, id := range ids {
		require.Equal(t, first, id, "concurrent bulk create %d must resolve to the same persisted row", i)
	}

	var count int
	err := testDB.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM contents WHERE content_hash = $1 AND original_url = $2`,
		"concurrent-email-hash", emailURL,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "concurrent deliveries of identical email content must produce exactly one row")
}
