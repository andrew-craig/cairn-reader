// Package sync provides functionality for synchronizing feed sources
// from external sources like the Kagi Small Web collection.
// It runs daily to ensure the feed database stays up-to-date with
// new feed sources.
package sync

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
)

// FeedSyncer fetches and updates the Kagi feed list daily.
// The Kagi Small Web collection is a curated list of high-quality,
// independent web feeds that forms the basis of content discovery.
type FeedSyncer struct {
	repo       db.FeedRepositoryInterface
	kagiURL    string
	httpClient *http.Client
}

// NewFeedSyncer creates a new FeedSyncer
func NewFeedSyncer(repo db.FeedRepositoryInterface, kagiURL string) *FeedSyncer {
	return &FeedSyncer{
		repo:    repo,
		kagiURL: kagiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Run starts the daily sync process that keeps the feed database updated.
// Syncs immediately on startup to ensure feeds are available right away,
// then runs every 24 hours to pick up new feeds from the Kagi collection.
// Non-fatal errors are logged but don't stop the sync loop.
func (s *FeedSyncer) Run(ctx context.Context) error {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Sync immediately on startup to ensure feeds are available
	if err := s.syncFeeds(ctx); err != nil {
		slog.Error("initial feed sync failed", slog.Any("error", err))
		// Continue despite error - will retry in 24 hours
	}

	for {
		select {
		case <-ticker.C:
			if err := s.syncFeeds(ctx); err != nil {
				slog.Error("feed sync failed", slog.Any("error", err))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// SyncOnce runs a single sync operation.
// Exposed for manual triggering via the /feeds/sync HTTP endpoint
// and for use in tests.
func (s *FeedSyncer) SyncOnce(ctx context.Context) error {
	return s.syncFeeds(ctx)
}

// syncFeeds performs the actual synchronization:
// 1. Fetches the feed list from the Kagi Small Web URL
// 2. Parses the list to extract feed URLs
// 3. Imports new feeds to the database (existing feeds are unchanged)
// 4. Logs statistics about the feed database state
func (s *FeedSyncer) syncFeeds(ctx context.Context) error {
	slog.Info("starting feed sync", slog.String("source_url", s.kagiURL))

	// Fetch the feed list from Kagi Small Web repository
	req, err := http.NewRequestWithContext(ctx, "GET", s.kagiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch kagi feed list: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", slog.Any("error", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Parse the feed list (one URL per line, # for comments)
	feedURLs := s.parseFeedList(string(body))
	if len(feedURLs) == 0 {
		return fmt.Errorf("no feed URLs found in response")
	}

	// Import feeds to database (uses ON CONFLICT DO NOTHING to avoid duplicates)
	if err := s.repo.ImportFeeds(ctx, feedURLs); err != nil {
		return fmt.Errorf("import feeds: %w", err)
	}

	// Log feed database statistics for monitoring
	total, enabled, disabled, neverFetched, err := s.repo.GetFeedStats(ctx)
	if err != nil {
		slog.Error("failed to get feed stats", slog.Any("error", err))
	} else {
		slog.Info("feed sync complete",
			slog.Int("total", total),
			slog.Int("enabled", enabled),
			slog.Int("disabled", disabled),
			slog.Int("never_fetched", neverFetched))
	}

	return nil
}

// parseFeedList parses the Kagi feed list format.
// Format:
//   - One URL per line
//   - Lines starting with # are comments
//   - Empty lines are ignored
//   - Only http:// and https:// URLs are accepted
func (s *FeedSyncer) parseFeedList(content string) []string {
	lines := strings.Split(content, "\n")
	var feedURLs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comment lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Only accept valid HTTP(S) URLs for security
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			feedURLs = append(feedURLs, line)
		}
	}

	return feedURLs
}
