//go:build integration
// +build integration

package service

import (
	"context"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestContentService_StoresAndReplacesSource_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := repository.NewContentRepository(testDB.DB)
	svc := NewContentService(repo, testDB.DB)
	ctx := context.Background()

	const url = "https://example.com/article"
	raw := "<html><head><title>A</title></head><body><article><p>Body one is long enough.</p></article></body></html>"

	// Create stores the raw input.
	created, err := svc.CreateFromHTML(ctx, url, raw, models.SourceTypeWeb, nil, nil)
	require.NoError(t, err)
	src, err := repo.GetSource(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, src)
	require.Equal(t, raw, src.RawHTML)
	require.False(t, src.ReadabilityFailed)

	// A dedup hit (same sanitized output, different raw bytes) returns the
	// existing row and leaves its source alone.
	rawVariant := raw + "<!-- tracking pixel variant -->"
	again, err := svc.CreateFromHTML(ctx, url, rawVariant, models.SourceTypeWeb, nil, nil)
	require.NoError(t, err)
	require.Equal(t, created.ID, again.ID, "variant must dedup against the first row")
	src, err = repo.GetSource(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, raw, src.RawHTML)

	// UpdateContent replaces it.
	updatedRaw := "<html><head><title>B</title></head><body><article><p>Body two replaced.</p></article></body></html>"
	_, err = svc.UpdateContent(ctx, created.ID, url, updatedRaw, nil)
	require.NoError(t, err)
	src, err = repo.GetSource(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, updatedRaw, src.RawHTML)
}

func TestContentService_BulkCreate_StoresSourcesForNewItemsOnly_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := repository.NewContentRepository(testDB.DB)
	svc := NewContentService(repo, testDB.DB)
	ctx := context.Background()
	feedID := uuid.New()

	seen := BulkContentItem{
		URL:          "https://example.com/seen",
		HTML:         "<html><head><title>Seen</title></head><body>Seen content body.</body></html>",
		SourceType:   models.SourceTypeRSS,
		SourceFeedID: &feedID,
	}
	first, _, err := svc.BulkCreateFromHTML(ctx, []BulkContentItem{seen})
	require.NoError(t, err)

	fresh := BulkContentItem{
		URL:          "https://example.com/fresh",
		HTML:         "<html><head><title>Fresh</title></head><body>Fresh content body.</body></html>",
		SourceType:   models.SourceTypeRSS,
		SourceFeedID: &feedID,
	}
	both, _, err := svc.BulkCreateFromHTML(ctx, []BulkContentItem{seen, fresh})
	require.NoError(t, err)
	require.Len(t, both, 2)

	var n int
	require.NoError(t, testDB.DB.QueryRowContext(ctx, `SELECT count(*) FROM content_sources`).Scan(&n))
	require.Equal(t, 2, n)

	src, err := repo.GetSource(ctx, first[0].ID)
	require.NoError(t, err)
	require.Equal(t, seen.HTML, src.RawHTML)
	for _, c := range both {
		if c.ID != first[0].ID {
			src, err := repo.GetSource(ctx, c.ID)
			require.NoError(t, err)
			require.Equal(t, fresh.HTML, src.RawHTML)
		}
	}
}
