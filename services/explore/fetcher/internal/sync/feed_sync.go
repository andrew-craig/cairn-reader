package sync

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/andrew-craig/cairn-explore/fetcher/internal/db"
)

// FeedSyncer fetches and updates the Kagi feed list daily
type FeedSyncer struct {
	repo       *db.FeedRepository
	kagiURL    string
	httpClient *http.Client
}

// NewFeedSyncer creates a new FeedSyncer
func NewFeedSyncer(repo *db.FeedRepository, kagiURL string) *FeedSyncer {
	return &FeedSyncer{
		repo:    repo,
		kagiURL: kagiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Run starts the daily sync process
func (s *FeedSyncer) Run(ctx context.Context) error {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Fetch immediately on startup, then once per day
	if err := s.syncFeeds(ctx); err != nil {
		log.Printf("Initial feed sync failed: %v", err)
		// Continue despite error - will retry in 24 hours
	}

	for {
		select {
		case <-ticker.C:
			if err := s.syncFeeds(ctx); err != nil {
				log.Printf("Feed sync failed: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// SyncOnce runs a single sync operation (useful for testing)
func (s *FeedSyncer) SyncOnce(ctx context.Context) error {
	return s.syncFeeds(ctx)
}

func (s *FeedSyncer) syncFeeds(ctx context.Context) error {
	log.Printf("Starting feed sync from %s", s.kagiURL)

	// Fetch from Kagi Small Web feed list
	req, err := http.NewRequestWithContext(ctx, "GET", s.kagiURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch kagi feed list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Parse feed URLs (one per line)
	feedURLs := s.parseFeedList(string(body))
	if len(feedURLs) == 0 {
		return fmt.Errorf("no feed URLs found in response")
	}

	// Import feeds to database
	if err := s.repo.ImportFeeds(ctx, feedURLs); err != nil {
		return fmt.Errorf("import feeds: %w", err)
	}

	// Log statistics
	total, enabled, disabled, neverFetched, err := s.repo.GetFeedStats(ctx)
	if err != nil {
		log.Printf("Failed to get feed stats: %v", err)
	} else {
		log.Printf("Feed sync complete - Total: %d, Enabled: %d, Disabled: %d, Never fetched: %d",
			total, enabled, disabled, neverFetched)
	}

	return nil
}

// parseFeedList parses the Kagi feed list format (one URL per line, # for comments)
func (s *FeedSyncer) parseFeedList(content string) []string {
	lines := strings.Split(content, "\n")
	var feedURLs []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Basic validation: must start with http:// or https://
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			feedURLs = append(feedURLs, line)
		}
	}

	return feedURLs
}
