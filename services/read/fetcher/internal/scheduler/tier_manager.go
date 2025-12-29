package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-read/fetcher/internal/repository"
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
	log.Printf("Tier manager started (update_interval=%v)", tm.config.UpdateInterval)
}

// Stop gracefully stops the tier manager
func (tm *TierManager) Stop() {
	close(tm.stopCh)
	tm.wg.Wait()
	log.Println("Tier manager stopped")
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

	log.Println("Starting tier update job")
	startTime := time.Now()

	// Get all active feeds
	feeds, err := tm.feedRepo.GetFeedsForTierUpdate(ctx)
	if err != nil {
		log.Printf("Error fetching feeds for tier update: %v", err)
		return
	}

	if len(feeds) == 0 {
		log.Println("No active feeds to update")
		return
	}

	log.Printf("Evaluating tiers for %d active feeds", len(feeds))

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
			log.Printf("Error updating tier for feed %s: %v", feed.ID, err)
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

		log.Printf("Feed %s tier changed: %s -> %s (last_published: %v)",
			feed.ID, oldTier, newTier, formatTimePtr(feed.LastPublishedAt))
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
	log.Printf("Tier update completed in %v", duration)
	log.Printf("  Total feeds: %d", stats.Total)
	log.Printf("  Updated: %d", stats.Updated)
	log.Printf("  Errors: %d", stats.Errors)
	log.Printf("  Tier distribution:")
	log.Printf("    Active: %d", stats.ByTier[models.PollingTierActive])
	log.Printf("    Moderate: %d", stats.ByTier[models.PollingTierModerate])
	log.Printf("    Quiet: %d", stats.ByTier[models.PollingTierQuiet])

	if len(stats.ByChange) > 0 {
		log.Println("  Tier changes:")
		for change, count := range stats.ByChange {
			log.Printf("    %s: %d", change, count)
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
