package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/worker"
	"github.com/google/uuid"
)

// PollSchedulerConfig holds configuration for the poll scheduler
type PollSchedulerConfig struct {
	// BatchSize is the number of feeds to fetch per batch
	BatchSize int
	// PollInterval is how often to check for feeds due for polling
	PollInterval time.Duration
	// WorkerPoolSize is the number of concurrent workers
	WorkerPoolSize int
}

// DefaultPollSchedulerConfig returns default configuration
func DefaultPollSchedulerConfig() *PollSchedulerConfig {
	return &PollSchedulerConfig{
		BatchSize:      50,
		PollInterval:   1 * time.Minute,
		WorkerPoolSize: 5,
	}
}

// PollScheduler manages feed polling with tiered strategy
type PollScheduler struct {
	config     *PollSchedulerConfig
	feedRepo   repository.FeedRepository
	feedWorker *worker.FeedWorker
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewPollScheduler creates a new poll scheduler
func NewPollScheduler(
	config *PollSchedulerConfig,
	feedRepo repository.FeedRepository,
	feedWorker *worker.FeedWorker,
) *PollScheduler {
	if config == nil {
		config = DefaultPollSchedulerConfig()
	}

	return &PollScheduler{
		config:     config,
		feedRepo:   feedRepo,
		feedWorker: feedWorker,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the polling scheduler
func (s *PollScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	slog.Info("Poll scheduler started",
		"batch_size", s.config.BatchSize,
		"poll_interval", s.config.PollInterval,
		"workers", s.config.WorkerPoolSize)
}

// Stop gracefully stops the polling scheduler
func (s *PollScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Poll scheduler stopped")
}

// run is the main polling loop
func (s *PollScheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	// Run immediately on start
	s.pollFeeds()

	for {
		select {
		case <-ticker.C:
			s.pollFeeds()
		case <-s.stopCh:
			return
		}
	}
}

// pollFeeds fetches feeds due for polling and processes them
func (s *PollScheduler) pollFeeds() {
	ctx := context.Background()

	// Get feeds due for polling
	feeds, err := s.feedRepo.GetFeedsDueForPolling(ctx, s.config.BatchSize)
	if err != nil {
		slog.Error("Error fetching feeds due for polling", "error", err)
		return
	}

	if len(feeds) == 0 {
		return
	}

	slog.Info("Found feeds due for polling", "count", len(feeds))

	// Process each feed using the worker pool. Each feed was already
	// atomically claimed by GetFeedsDueForPolling; if Submit couldn't queue
	// it (pool stopping, or queue full), release the claim so it's retried
	// on the next poll instead of sitting claimed for the full lease.
	for _, feed := range feeds {
		if !s.feedWorker.Submit(feed) {
			if err := s.feedRepo.ReleaseClaim(ctx, feed.ID); err != nil {
				slog.Error("Error releasing feed claim", "feed_id", feed.ID, "error", err)
			}
		}
	}
}

// GetPollingInterval returns the polling interval for a given tier
func GetPollingInterval(tier models.PollingTier) time.Duration {
	switch tier {
	case models.PollingTierActive:
		return 1 * time.Hour
	case models.PollingTierModerate:
		return 6 * time.Hour
	case models.PollingTierQuiet:
		return 24 * time.Hour
	default:
		return 6 * time.Hour
	}
}

// CalculateNextPollTime calculates the next poll time based on the current tier
func CalculateNextPollTime(tier models.PollingTier) time.Time {
	return time.Now().Add(GetPollingInterval(tier))
}

// PromoteTier promotes a feed to a higher tier when new content is published
func PromoteTier(currentTier models.PollingTier) models.PollingTier {
	switch currentTier {
	case models.PollingTierQuiet:
		return models.PollingTierModerate
	case models.PollingTierModerate:
		return models.PollingTierActive
	case models.PollingTierActive:
		return models.PollingTierActive // Already at highest tier
	default:
		return currentTier
	}
}

// DetermineTierByActivity determines the appropriate tier based on last published time
func DetermineTierByActivity(lastPublishedAt *time.Time) models.PollingTier {
	if lastPublishedAt == nil {
		// No content published yet - use moderate tier as default
		return models.PollingTierModerate
	}

	daysSincePublished := time.Since(*lastPublishedAt).Hours() / 24

	switch {
	case daysSincePublished <= 7:
		return models.PollingTierActive
	case daysSincePublished <= 30:
		return models.PollingTierModerate
	default:
		return models.PollingTierQuiet
	}
}

// UpdateFeedPollingAfterSuccess updates feed polling info after a successful fetch
func UpdateFeedPollingAfterSuccess(
	ctx context.Context,
	feedRepo repository.FeedRepository,
	feedID uuid.UUID,
	currentTier models.PollingTier,
	foundNewItems bool,
) error {
	now := time.Now()
	var newTier models.PollingTier

	if foundNewItems {
		// Promote tier when new content is found
		newTier = PromoteTier(currentTier)
	} else {
		// Keep current tier if no new items
		newTier = currentTier
	}

	nextPollAt := CalculateNextPollTime(newTier)

	var lastPublishedAt *time.Time
	if foundNewItems {
		lastPublishedAt = &now
	} else {
		lastPublishedAt = nil
	}

	return feedRepo.UpdatePollingInfo(ctx, feedID, &now, lastPublishedAt, nextPollAt, newTier)
}

// UpdateFeedPollingAfterError updates feed after a failed fetch attempt
func UpdateFeedPollingAfterError(
	ctx context.Context,
	feedRepo repository.FeedRepository,
	feedID uuid.UUID,
	currentTier models.PollingTier,
	consecutiveErrorDays int,
	errorMessage string,
) error {
	now := time.Now()

	// Update error info
	if err := feedRepo.UpdateErrorInfo(ctx, feedID, consecutiveErrorDays, &now, &errorMessage); err != nil {
		return fmt.Errorf("failed to update error info: %w", err)
	}

	// If 7 consecutive error days, disable the feed
	if consecutiveErrorDays >= 7 {
		if err := feedRepo.UpdateStatus(ctx, feedID, models.FeedStatusDisabled); err != nil {
			return fmt.Errorf("failed to disable feed: %w", err)
		}
		slog.Warn("Feed disabled after consecutive error days", "feed_id", feedID, "error_days", consecutiveErrorDays)
	}

	return nil
}
