// Package sync provides functionality for synchronizing feed sources
// from a configured feed list. The list may be supplied as a local file
// (preferred, allowing self-hosters to curate their own feeds) or fetched
// from an HTTP URL (a default list maintained in this repository).
// It runs daily so user edits to a mounted file, or upstream updates to
// the default list, are picked up automatically.
package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cairn-app/cairn-reader/services/explore/fetcher/internal/db"
)

// FeedSyncer loads feed URLs from a curated list and imports them into
// the fetcher's feed database. The list is resolved as follows:
//  1. If listPath is set and the file exists, read feed URLs from it.
//     This is the path operators mount their own list at.
//  2. Otherwise, HTTP GET listURL and use the response body. This is
//     typically the default list maintained in the Cairn repository.
type FeedSyncer struct {
	repo       db.FeedRepositoryInterface
	listPath   string
	listURL    string
	httpClient *http.Client
}

// NewFeedSyncer creates a FeedSyncer that prefers listPath on disk and
// falls back to downloading listURL when the file is missing.
// listPath may be empty to force URL-only loading.
func NewFeedSyncer(repo db.FeedRepositoryInterface, listPath, listURL string) *FeedSyncer {
	return &FeedSyncer{
		repo:     repo,
		listPath: listPath,
		listURL:  listURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Run starts the daily sync process that keeps the feed database updated.
// Syncs immediately on startup to ensure feeds are available right away,
// then runs every 24 hours to pick up edits to a mounted list file or
// updates to the default list in the repository.
// Non-fatal errors are logged but don't stop the sync loop.
func (s *FeedSyncer) Run(ctx context.Context) error {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	if err := s.syncFeeds(ctx); err != nil {
		slog.Error("initial feed sync failed", slog.Any("error", err))
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
// 1. Loads the feed list from the local file or URL
// 2. Parses the list to extract feed URLs
// 3. Imports new feeds to the database (existing feeds are unchanged)
// 4. Logs statistics about the feed database state
func (s *FeedSyncer) syncFeeds(ctx context.Context) error {
	body, source, err := s.loadFeedList(ctx)
	if err != nil {
		return fmt.Errorf("load feed list: %w", err)
	}
	slog.Info("loaded feed list", slog.String("source", source), slog.Int("bytes", len(body)))

	feedURLs := parseFeedList(string(body))
	if len(feedURLs) == 0 {
		return fmt.Errorf("no feed URLs found in %s", source)
	}

	if err := s.repo.ImportFeeds(ctx, feedURLs); err != nil {
		return fmt.Errorf("import feeds: %w", err)
	}

	total, enabled, disabled, neverFetched, err := s.repo.GetFeedStats(ctx)
	if err != nil {
		slog.Error("failed to get feed stats", slog.Any("error", err))
	} else {
		slog.Info("feed sync complete",
			slog.String("source", source),
			slog.Int("parsed", len(feedURLs)),
			slog.Int("total", total),
			slog.Int("enabled", enabled),
			slog.Int("disabled", disabled),
			slog.Int("never_fetched", neverFetched))
	}

	return nil
}

// loadFeedList resolves the feed list from disk (if present) or the
// configured URL. It returns the raw bytes plus a human-readable source
// label for logging.
func (s *FeedSyncer) loadFeedList(ctx context.Context) ([]byte, string, error) {
	if s.listPath != "" {
		body, err := os.ReadFile(s.listPath)
		if err == nil {
			return body, "file:" + s.listPath, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", fmt.Errorf("read feed list file %s: %w", s.listPath, err)
		}
		slog.Info("feed list file not present, falling back to URL",
			slog.String("path", s.listPath),
			slog.String("url", s.listURL))
	}

	if s.listURL == "" {
		return nil, "", fmt.Errorf("no feed list available: path %q missing and no URL configured", s.listPath)
	}

	body, err := s.downloadFeedList(ctx)
	if err != nil {
		return nil, "", err
	}
	return body, "url:" + s.listURL, nil
}

func (s *FeedSyncer) downloadFeedList(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch feed list: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", slog.Any("error", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, s.listURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

// parseFeedList parses a feed list.
// Format:
//   - One URL per line
//   - Lines starting with # are comments
//   - Empty lines are ignored
//   - Only http:// and https:// URLs are accepted
func parseFeedList(content string) []string {
	lines := strings.Split(content, "\n")
	var feedURLs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			feedURLs = append(feedURLs, line)
		}
	}

	return feedURLs
}
