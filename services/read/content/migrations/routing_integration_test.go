//go:build integration
// +build integration

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestFeedReadsRoutingMigration_UpDown_Integration applies 000005 on top of a
// database that already holds user_contents rows: the new list column must
// backfill existing rows to 'reads', the constraints must reject bad values,
// and the down migration must restore the previous schema.
func TestFeedReadsRoutingMigration_UpDown_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	connInfo := testutil.GetTestConnectionInfo()
	dbName := fmt.Sprintf("cairn_migration_routing_test_%d", time.Now().UnixNano())

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

	apply := func(name string) {
		t.Helper()
		sqlBytes, err := FS.ReadFile(name)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, string(sqlBytes))
		require.NoError(t, err, "applying %s", name)
	}
	for _, name := range []string{
		"000001_initial_schema.up.sql",
		"000002_richer_status_vocabulary.up.sql",
		"000003_scroll_position_fraction.up.sql",
		"000004_unique_content_dedup.up.sql",
	} {
		apply(name)
	}

	// A row that predates the Feed/Reads split.
	userID := uuid.New()
	var contentID uuid.UUID
	require.NoError(t, db.QueryRowContext(ctx, `
		INSERT INTO contents (id, content_hash, cleaned_html, original_url, title, source_type)
		VALUES ($1, 'h', '<p>x</p>', 'https://example.com/a', 'A', 'web') RETURNING id`, uuid.New()).Scan(&contentID))
	_, err = db.ExecContext(ctx, `INSERT INTO user_contents (user_id, content_id) VALUES ($1, $2)`, userID, contentID)
	require.NoError(t, err)

	apply("000005_feed_reads_routing.up.sql")

	var list string
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT list FROM user_contents WHERE user_id = $1`, userID).Scan(&list))
	require.Equal(t, "reads", list, "existing rows must backfill to reads")

	_, err = db.ExecContext(ctx, `UPDATE user_contents SET list = 'explore' WHERE user_id = $1`, userID)
	require.Error(t, err, "list must be constrained to feed|reads")

	feedKey := uuid.New()
	_, err = db.ExecContext(ctx, `
		INSERT INTO source_routes (user_id, source_type, source_key, list) VALUES ($1, 'rss', $2, 'feed')`,
		userID, feedKey)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO source_routes (user_id, source_type, source_key, list) VALUES ($1, 'rss', $2, 'reads')`,
		userID, feedKey)
	require.Error(t, err, "(user_id, source_type, source_key) must be unique")

	_, err = db.ExecContext(ctx, `
		INSERT INTO source_routes (user_id, source_type, source_key, list) VALUES ($1, 'web', $2, 'feed')`,
		userID, uuid.New())
	require.Error(t, err, "source_type must be rss|email")

	apply("000005_feed_reads_routing.down.sql")

	var n int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.columns
		WHERE (table_name = 'user_contents' AND column_name = 'list')
		   OR (table_name = 'contents' AND column_name = 'source_sender_id')`).Scan(&n))
	require.Zero(t, n, "down must drop the new columns")
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_name = 'source_routes'`).Scan(&n))
	require.Zero(t, n, "down must drop source_routes")

	// And it must be re-appliable.
	apply("000005_feed_reads_routing.up.sql")
}
