//go:build integration
// +build integration

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestUniqueContentDedupBackfill_UserOwnsTwoLosersMappingToSameSurvivor_Integration
// reproduces a defect caught in review of 000004_unique_content_dedup: the
// backfill originally re-pointed user_contents from losers to the survivor
// with a plain UPDATE guarded by "NOT EXISTS (already owns the survivor)".
// That guard is evaluated once against the statement's pre-update snapshot,
// so when a single user owns two different loser rows that both map to the
// same survivor — exactly the shape the concurrent-duplication bug being
// fixed produces — both updates pass the guard and then collide with each
// other on unique_user_content, aborting the whole migration on real
// duplicate data. The fix reassigns via INSERT ... ON CONFLICT DO NOTHING
// followed by an unconditional DELETE of loser links instead.
func TestUniqueContentDedupBackfill_UserOwnsTwoLosersMappingToSameSurvivor_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	connInfo := testutil.GetTestConnectionInfo()
	dbName := fmt.Sprintf("cairn_migration_backfill_test_%d", time.Now().UnixNano())

	adminDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		connInfo.Host, connInfo.Port, connInfo.User, connInfo.Password, connInfo.SSLMode)
	adminDB, err := sql.Open("postgres", adminDSN)
	require.NoError(t, err)
	defer func() { _ = adminDB.Close() }()

	_, err = adminDB.Exec("CREATE DATABASE " + dbName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = adminDB.Exec(fmt.Sprintf(
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()", dbName))
		_, _ = adminDB.Exec("DROP DATABASE IF EXISTS " + dbName)
	})

	testDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		connInfo.Host, connInfo.Port, connInfo.User, connInfo.Password, dbName, connInfo.SSLMode)
	db, err := sql.Open("postgres", testDSN)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	require.NoError(t, db.PingContext(ctx))

	// Apply every migration up to (but not including) the one under test.
	for _, name := range []string{
		"000001_initial_schema.up.sql",
		"000002_richer_status_vocabulary.up.sql",
		"000003_scroll_position_fraction.up.sql",
	} {
		sqlBytes, err := FS.ReadFile(name)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(sqlBytes))
		require.NoError(t, err, "applying %s", name)
	}

	// Seed three RSS rows sharing (content_hash, source_feed_id) — survivor
	// a (earliest), losers b and c — with a single user who saved BOTH
	// losers and never the survivor directly.
	feedID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	insertContent := `
		INSERT INTO contents (id, content_hash, cleaned_html, original_url, title, source_type, source_feed_id, created_at)
		VALUES ($1, 'hash-dup', '<p>x</p>', $2, 'X', 'rss', $3, $4)
		RETURNING id`

	var survivorID, loserB, loserC uuid.UUID
	require.NoError(t, db.QueryRowContext(ctx, insertContent,
		uuid.New(), "https://example.com/1", feedID, now.Add(-3*time.Hour)).Scan(&survivorID))
	require.NoError(t, db.QueryRowContext(ctx, insertContent,
		uuid.New(), "https://example.com/2", feedID, now.Add(-2*time.Hour)).Scan(&loserB))
	require.NoError(t, db.QueryRowContext(ctx, insertContent,
		uuid.New(), "https://example.com/3", feedID, now.Add(-1*time.Hour)).Scan(&loserC))

	_, err = db.ExecContext(ctx, `INSERT INTO user_contents (user_id, content_id) VALUES ($1, $2)`, userID, loserB)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO user_contents (user_id, content_id) VALUES ($1, $2)`, userID, loserC)
	require.NoError(t, err)

	// Applying the migration under test must not fail.
	upSQL, err := FS.ReadFile("000004_unique_content_dedup.up.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(upSQL))
	require.NoError(t, err, "backfill must not fail when a user owns two losers mapping to the same survivor")

	var contentCount int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM contents WHERE content_hash = 'hash-dup'`).Scan(&contentCount))
	require.Equal(t, 1, contentCount, "duplicate content rows must collapse to the survivor")

	var linkCount int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM user_contents WHERE user_id = $1`, userID).Scan(&linkCount))
	require.Equal(t, 1, linkCount, "the user must end up with exactly one link, not two")

	var linkedContentID uuid.UUID
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT content_id FROM user_contents WHERE user_id = $1`, userID).Scan(&linkedContentID))
	require.Equal(t, survivorID, linkedContentID, "the surviving link must point at the survivor")

	// The down migration must also apply cleanly.
	downSQL, err := FS.ReadFile("000004_unique_content_dedup.down.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(downSQL))
	require.NoError(t, err)
}
