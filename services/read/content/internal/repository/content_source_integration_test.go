//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newSourcedContent(hash, url, raw string, failed bool) *models.Content {
	return &models.Content{
		ContentHash: hash,
		CleanedHTML: "<p>cleaned</p>",
		OriginalURL: url,
		Title:       "Title " + hash,
		SourceType:  models.SourceTypeWeb,
		Source:      &models.ContentSource{RawHTML: raw, ReadabilityFailed: failed},
	}
}

func TestContentRepository_Create_StoresSource_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	c := newSourcedContent("h1", "https://example.com/1", "<html>raw</html>", true)
	require.NoError(t, repo.Create(ctx, c))

	src, err := repo.GetSource(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, src)
	require.Equal(t, "<html>raw</html>", src.RawHTML)
	require.True(t, src.ReadabilityFailed)
	require.Equal(t, c.ID, src.ContentID)
}

func TestContentRepository_GetSource_Absent_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)

	src, err := repo.GetSource(context.Background(), uuid.New())
	require.NoError(t, err)
	require.Nil(t, src)
}

func TestContentRepository_Create_DedupHitKeepsOriginalSource_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	first := newSourcedContent("h1", "https://example.com/1", "<html>first</html>", false)
	require.NoError(t, repo.Create(ctx, first))

	// Same dedup key: the insert resolves to the first row.
	second := newSourcedContent("h1", "https://example.com/1", "<html>second</html>", true)
	require.NoError(t, repo.Create(ctx, second))
	require.Equal(t, first.ID, second.ID)

	src, err := repo.GetSource(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "<html>first</html>", src.RawHTML)
	require.False(t, src.ReadabilityFailed)
}

func TestContentRepository_Update_ReplacesSource_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	c := newSourcedContent("h1", "https://example.com/1", "<html>old</html>", false)
	require.NoError(t, repo.Create(ctx, c))
	old, err := repo.GetSource(ctx, c.ID)
	require.NoError(t, err)

	c.ContentHash = "h2"
	c.Source = &models.ContentSource{RawHTML: "<html>new</html>", ReadabilityFailed: true}
	require.NoError(t, repo.Update(ctx, c))

	src, err := repo.GetSource(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, "<html>new</html>", src.RawHTML)
	require.True(t, src.ReadabilityFailed)
	require.True(t, src.CreatedAt.After(old.CreatedAt), "replacing restarts the retention clock")

	// An update without a source leaves the stored one alone.
	c.Source = nil
	c.Title = "renamed"
	require.NoError(t, repo.Update(ctx, c))
	src, err = repo.GetSource(ctx, c.ID)
	require.NoError(t, err)
	require.Equal(t, "<html>new</html>", src.RawHTML)
}

func TestContentRepository_BulkCreate_StoresSourcesOnlyForInsertedRows_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	existing := newSourcedContent("h1", "https://example.com/1", "<html>existing</html>", false)
	require.NoError(t, repo.Create(ctx, existing))

	dupOfExisting := newSourcedContent("h1", "https://example.com/1", "<html>dup</html>", true)
	fresh := newSourcedContent("h2", "https://example.com/2", "<html>fresh</html>", true)
	freshRepeat := newSourcedContent("h2", "https://example.com/2", "<html>fresh-repeat</html>", false)
	require.NoError(t, repo.BulkCreate(ctx, []*models.Content{dupOfExisting, fresh, freshRepeat}))

	require.Equal(t, existing.ID, dupOfExisting.ID)
	require.Equal(t, fresh.ID, freshRepeat.ID)

	src, err := repo.GetSource(ctx, existing.ID)
	require.NoError(t, err)
	require.Equal(t, "<html>existing</html>", src.RawHTML, "a dedup hit must not overwrite the source")

	src, err = repo.GetSource(ctx, fresh.ID)
	require.NoError(t, err)
	require.Equal(t, "<html>fresh</html>", src.RawHTML)
	require.True(t, src.ReadabilityFailed)

	var n int
	require.NoError(t, testDB.DB.QueryRowContext(ctx, `SELECT count(*) FROM content_sources`).Scan(&n))
	require.Equal(t, 2, n)
}

func TestContentRepository_DeleteExpiredSources_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	var ids []uuid.UUID
	for i, hash := range []string{"old1", "old2", "old3", "new"} {
		c := newSourcedContent(hash, "https://example.com/"+hash, "<html>"+hash+"</html>", false)
		require.NoError(t, repo.Create(ctx, c))
		ids = append(ids, c.ID)
		if i < 3 {
			_, err := testDB.DB.ExecContext(ctx,
				`UPDATE content_sources SET created_at = $1 WHERE content_id = $2`,
				time.Now().Add(-40*24*time.Hour), c.ID)
			require.NoError(t, err)
		}
	}
	retention := 30 * 24 * time.Hour

	n, err := repo.DeleteExpiredSources(ctx, retention, 2)
	require.NoError(t, err)
	require.EqualValues(t, 2, n, "a single call is bounded by batchSize")
	n, err = repo.DeleteExpiredSources(ctx, retention, 2)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	n, err = repo.DeleteExpiredSources(ctx, retention, 2)
	require.NoError(t, err)
	require.EqualValues(t, 0, n)

	src, err := repo.GetSource(ctx, ids[3])
	require.NoError(t, err)
	require.NotNil(t, src, "recent source survives")

	// Contents themselves are untouched by source expiry.
	for _, id := range ids {
		_, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
	}
}

func TestContentRepository_DeleteContentCascadesToSource_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)
	repo := NewContentRepository(testDB.DB)
	ctx := context.Background()

	c := newSourcedContent("h1", "https://example.com/1", "<html>raw</html>", false)
	require.NoError(t, repo.Create(ctx, c))

	_, err := testDB.DB.ExecContext(ctx, `DELETE FROM contents WHERE id = $1`, c.ID)
	require.NoError(t, err)

	src, err := repo.GetSource(ctx, c.ID)
	require.NoError(t, err)
	require.Nil(t, src)
}
