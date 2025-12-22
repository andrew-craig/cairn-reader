package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use test database connection
	connStr := "host=localhost port=5433 user=fetcher password=fetcher_password dbname=fetcher_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// CleanupTestDB cleans up test database tables
func CleanupTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	// Delete in correct order due to foreign keys
	queries := []string{
		"DELETE FROM fetch_history",
		"DELETE FROM feeds",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}
}

// CreateTestFeed inserts a test feed into the database
func CreateTestFeed(t *testing.T, db *sql.DB, url string, enabled bool, lastFetched *time.Time, failures int) int {
	t.Helper()

	query := `
		INSERT INTO feeds (url, title, description, last_fetched_at, consecutive_failures, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var feedID int
	err := db.QueryRow(
		query,
		url,
		fmt.Sprintf("Test Feed: %s", url),
		"Test feed description",
		lastFetched,
		failures,
		enabled,
	).Scan(&feedID)

	if err != nil {
		t.Fatalf("Failed to create test feed: %v", err)
	}

	return feedID
}

// GetFeedByID retrieves a feed by ID
func GetFeedByID(t *testing.T, db *sql.DB, feedID int) (url string, enabled bool, failures int, lastFetched *time.Time) {
	t.Helper()

	query := "SELECT url, enabled, consecutive_failures, last_fetched_at FROM feeds WHERE id = $1"
	err := db.QueryRow(query, feedID).Scan(&url, &enabled, &failures, &lastFetched)
	if err != nil {
		t.Fatalf("Failed to get feed: %v", err)
	}

	return
}

// CountFeeds returns the total number of feeds in the database
func CountFeeds(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count feeds: %v", err)
	}
	return count
}

// CountFetchHistory returns the total number of fetch history records
func CountFetchHistory(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM fetch_history").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count fetch history: %v", err)
	}
	return count
}

// WaitForCondition polls a condition until it returns true or times out
func WaitForCondition(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for condition")
		case <-ticker.C:
			if check() {
				return
			}
		}
	}
}

// Sample RSS feed URLs for testing
var (
	TestFeedURL1 = "https://example.com/feed1.xml"
	TestFeedURL2 = "https://example.com/feed2.xml"
	TestFeedURL3 = "https://example.com/feed3.xml"
	TestFeedURL4 = "https://example.com/feed4.xml"
	TestFeedURL5 = "https://example.com/feed5.xml"
)

// SampleRSSFeed returns a sample RSS feed XML for testing
func SampleRSSFeed(title string, itemCount int) string {
	items := ""
	for i := 1; i <= itemCount; i++ {
		items += fmt.Sprintf(`
		<item>
			<title>Article %d - %s</title>
			<link>https://example.com/article-%d</link>
			<description>Description for article %d</description>
			<pubDate>%s</pubDate>
			<author>Test Author</author>
			<guid>https://example.com/article-%d</guid>
		</item>
		`, i, title, i, i, time.Now().Add(-time.Duration(i)*time.Hour).Format(time.RFC1123Z), i)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>%s</title>
		<link>https://example.com</link>
		<description>Test RSS Feed</description>
		<language>en-us</language>
		<lastBuildDate>%s</lastBuildDate>
		%s
	</channel>
</rss>`, title, time.Now().Format(time.RFC1123Z), items)
}

// SampleKagiList returns a sample Kagi feed list for testing
func SampleKagiList() string {
	return `# Kagi Small Web Feed List
# Comment lines start with #
https://example.com/feed1.xml
https://example.com/feed2.xml
https://example.com/feed3.xml

# Empty lines are ignored
https://example.com/feed4.xml
https://example.com/feed5.xml
`
}
