package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
)

// TierManagerConfig holds configuration for the tier manager
type TierManagerConfig struct {
	// UpdateInterval is how often to run tier updates (default: daily)
	UpdateInterval time.Duration
}

// DefaultTierManagerConfig returns default configuration
func DefaultTierManagerConfig() *TierManagerConfig {
	return &TierManagerConfig{
		UpdateInterval: 24 * time.Hour,
	}
}

// TierManager manages feed polling tiers based on activity
type TierManager struct {
	config   *TierManagerConfig
	feedRepo repository.FeedRepository
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewTierManager creates a new tier manager
func NewTierManager(
	config *TierManagerConfig,
	feedRepo repository.FeedRepository,
) *TierManager {
	if config == nil {
		config = DefaultTierManagerConfig()
	}

	return &TierManager{
		config:   config,
		feedRepo: feedRepo,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the tier manager
func (tm *TierManager) Start() {
	tm.wg.Add(1)
	go tm.run()
	slog.Info("Tier manager started", "update_interval", tm.config.UpdateInterval)
}

// Stop gracefully stops the tier manager
func (tm *TierManager) Stop() {
	close(tm.stopCh)
	tm.wg.Wait()
	slog.Info("Tier manager stopped")
}

// run is the main tier management loop
func (tm *TierManager) run() {
	defer tm.wg.Done()

	ticker := time.NewTicker(tm.config.UpdateInterval)
	defer ticker.Stop()

	// Run immediately on start
	tm.updateTiers()

	for {
		select {
		case <-ticker.C:
			tm.updateTiers()
		case <-tm.stopCh:
			return
		}
	}
}

// updateTiers updates all feed tiers based on their activity
func (tm *TierManager) updateTiers() {
	ctx := context.Background()

	slog.Info("Starting tier update job")
	startTime := time.Now()

	// Get all active feeds
	feeds, err := tm.feedRepo.GetFeedsForTierUpdate(ctx)
	if err != nil {
		slog.Error("Error fetching feeds for tier update", "error", err)
		return
	}

	if len(feeds) == 0 {
		slog.Info("No active feeds to update")
		return
	}

	slog.Info("Evaluating tiers for active feeds", "count", len(feeds))

	// Track statistics
	stats := &TierUpdateStats{
		Total:    len(feeds),
		Updated:  0,
		Errors:   0,
		ByTier:   make(map[models.PollingTier]int),
		ByChange: make(map[string]int),
	}

	// Update each feed's tier
	for _, feed := range feeds {
		if err := tm.updateFeedTier(ctx, feed, stats); err != nil {
			slog.Error("Error updating tier for feed", "feed_id", feed.ID, "error", err)
			stats.Errors++
		}
	}

	duration := time.Since(startTime)
	tm.logTierUpdateStats(stats, duration)
}

// updateFeedTier updates a single feed's tier based on activity
func (tm *TierManager) updateFeedTier(
	ctx context.Context,
	feed *models.Feed,
	stats *TierUpdateStats,
) error {
	// Determine new tier based on last published time
	newTier := DetermineTierByActivity(feed.LastPublishedAt)

	// Track tier distribution
	stats.ByTier[newTier]++

	// Only update if tier changed
	if newTier != feed.PollingTier {
		oldTier := feed.PollingTier
		nextPollAt := CalculateNextPollTime(newTier)

		err := tm.feedRepo.UpdatePollingInfo(
			ctx,
			feed.ID,
			feed.LastFetchedAt,
			feed.LastPublishedAt,
			nextPollAt,
			newTier,
		)

		if err != nil {
			return err
		}

		stats.Updated++
		change := string(oldTier) + "->" + string(newTier)
		stats.ByChange[change]++

		slog.Info("Feed tier changed",
			"feed_id", feed.ID,
			"old_tier", oldTier,
			"new_tier", newTier,
			"last_published", formatTimePtr(feed.LastPublishedAt))
	}

	return nil
}

// TierUpdateStats tracks statistics for tier updates
type TierUpdateStats struct {
	Total    int
	Updated  int
	Errors   int
	ByTier   map[models.PollingTier]int
	ByChange map[string]int
}

// logTierUpdateStats logs summary statistics
func (tm *TierManager) logTierUpdateStats(stats *TierUpdateStats, duration time.Duration) {
	slog.Info("Tier update completed",
		"duration", duration,
		"total_feeds", stats.Total,
		"updated", stats.Updated,
		"errors", stats.Errors,
		"tier_active", stats.ByTier[models.PollingTierActive],
		"tier_moderate", stats.ByTier[models.PollingTierModerate],
		"tier_quiet", stats.ByTier[models.PollingTierQuiet])

	if len(stats.ByChange) > 0 {
		for change, count := range stats.ByChange {
			slog.Info("Tier change", "change", change, "count", count)
		}
	}
}

// formatTimePtr formats a time pointer for logging
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return t.Format(time.RFC3339)
}
