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

// TestBulkCreateFromHTML_MixedNewAndSeenBatch_Integration reproduces P2-C6: a
// re-polled RSS feed routinely redelivers an already-stored item alongside a
// genuinely new one in the same batch. Against real Postgres (no ON CONFLICT
// on the multi-row INSERT, and the pre-existing row carrying its real id),
// the mixed batch must not PK-collide and drop the whole batch — both items
// must survive, and only the new one must be inserted.
func TestBulkCreateFromHTML_MixedNewAndSeenBatch_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	contentRepo := repository.NewContentRepository(testDB.DB)
	svc := NewContentService(contentRepo, testDB.DB)

	ctx := context.Background()
	feedID := uuid.New()

	seenItem := BulkContentItem{
		URL:          "https://example.com/already-seen-article",
		HTML:         "<html><head><title>Already Seen</title></head><body>Seen content body.</body></html>",
		SourceType:   "rss",
		SourceFeedID: &feedID,
	}

	// First poll: item is genuinely new and gets stored for real.
	firstBatch, errs, err := svc.BulkCreateFromHTML(ctx, []BulkContentItem{seenItem})
	require.NoError(t, err)
	require.Empty(t, errs)
	require.Len(t, firstBatch, 1)
	storedID := firstBatch[0].ID
	require.NotEqual(t, uuid.Nil, storedID)

	// Second poll: the feed redelivers the same item (now "seen", carrying a
	// real DB id) mixed with a brand-new item in the same batch.
	newItem := BulkContentItem{
		URL:          "https://example.com/brand-new-article",
		HTML:         "<html><head><title>Brand New</title></head><body>New content body.</body></html>",
		SourceType:   "rss",
		SourceFeedID: &feedID,
	}

	mixedBatch, errs, err := svc.BulkCreateFromHTML(ctx, []BulkContentItem{seenItem, newItem})
	require.NoError(t, err, "mixed new+seen batch must not fail with a PK collision")
	require.Empty(t, errs)
	require.Len(t, mixedBatch, 2)

	// The re-delivered item must resolve back to the same stored row, not a
	// second insert attempt.
	require.Equal(t, storedID, mixedBatch[0].ID)

	// The new item must have actually been created.
	newContent, err := contentRepo.GetByContentHashAndFeedID(ctx, mixedBatch[1].ContentHash, feedID)
	require.NoError(t, err)
	require.NotNil(t, newContent, "new item in the mixed batch must be persisted")
	require.NotEqual(t, storedID, newContent.ID)
}
