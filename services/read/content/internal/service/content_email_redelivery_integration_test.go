//go:build integration
// +build integration

package service

import (
	"context"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestBulkCreateFromHTML_EmailRepeatDelivery_Integration reproduces the
// email/manual half of the dedup finding: BulkCreateFromHTML only checked
// for duplicates when SourceType=="rss", so an outbox retry after a partial
// delivery failure (the client succeeds server-side but never sees the
// response, or the follow-up add-to-user call fails) redelivered the exact
// same email payload — same content_hash, same synthetic
// "email://<raw_email_id>" URL, per email_processor_worker.go — and created
// a fresh duplicate content row every time. It must now dedupe like RSS does.
func TestBulkCreateFromHTML_EmailRepeatDelivery_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	contentRepo := repository.NewContentRepository(testDB.DB)
	svc := NewContentService(contentRepo, testDB.DB)

	ctx := context.Background()
	emailURL := "email://" + uuid.New().String()

	emailItem := BulkContentItem{
		URL:        emailURL,
		HTML:       "<html><head><title>Newsletter</title></head><body>Newsletter body.</body></html>",
		SourceType: "email",
	}

	// First delivery attempt: genuinely new, gets stored.
	firstBatch, errs, err := svc.BulkCreateFromHTML(ctx, []BulkContentItem{emailItem})
	require.NoError(t, err)
	require.Empty(t, errs)
	require.Len(t, firstBatch, 1)
	storedID := firstBatch[0].ID
	require.NotEqual(t, uuid.Nil, storedID)

	// Outbox retry: same raw_email_id -> same URL, same HTML -> same
	// content_hash, redelivered as its own bulk request.
	secondBatch, errs, err := svc.BulkCreateFromHTML(ctx, []BulkContentItem{emailItem})
	require.NoError(t, err, "repeat email delivery must not error")
	require.Empty(t, errs)
	require.Len(t, secondBatch, 1)
	require.Equal(t, storedID, secondBatch[0].ID, "retry must resolve to the original stored content, not a new row")

	var count int
	err = testDB.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM contents WHERE original_url = $1`, emailURL,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "repeat email delivery must not create a duplicate content row")
}
