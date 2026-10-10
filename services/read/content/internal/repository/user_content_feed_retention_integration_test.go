//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUserContentRepository_DeleteExpiredFeed_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	repo := NewUserContentRepository(testDB.DB)
	ctx := context.Background()
	userID := uuid.New()

	insert := func(name, list string, favorite bool, age time.Duration) uuid.UUID {
		contentID := uuid.New()
		_, err := testDB.DB.ExecContext(ctx, `
			INSERT INTO contents (id, content_hash, cleaned_html, original_url, title, source_type)
			VALUES ($1, $2, '<p>x</p>', $3, $2, 'web')
		`, contentID, name, "https://example.com/"+name)
		require.NoError(t, err)
		_, err = testDB.DB.ExecContext(ctx, `
			INSERT INTO user_contents (id, user_id, content_id, status, list, is_favorite, added_at)
			VALUES ($1, $2, $3, 'unread', $4, $5, $6)
		`, uuid.New(), userID, contentID, list, favorite, time.Now().Add(-age))
		require.NoError(t, err)
		return contentID
	}

	const old = 31 * 24 * time.Hour
	expired := insert("feed-expired", "feed", false, old)
	fresh := insert("feed-fresh", "feed", false, 24*time.Hour)
	favorited := insert("feed-favorited", "feed", true, old)
	read := insert("reads-old", "reads", false, old)

	deleted, err := repo.DeleteExpiredFeed(ctx, 30*24*time.Hour, 1000)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	exists := func(contentID uuid.UUID) bool {
		var n int
		require.NoError(t, testDB.DB.QueryRowContext(ctx,
			`SELECT count(*) FROM user_contents WHERE content_id = $1`, contentID).Scan(&n))
		return n == 1
	}
	require.False(t, exists(expired), "expired feed item must be deleted")
	require.True(t, exists(fresh), "recent feed item must be kept")
	require.True(t, exists(favorited), "favorited feed item must be kept")
	require.True(t, exists(read), "old reads item must be kept")
}
