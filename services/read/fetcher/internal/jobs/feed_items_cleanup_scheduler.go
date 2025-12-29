package jobs

import (
	"log"
	"sync"
	"time"
)

// FeedItemsCleanupSchedulerConfig holds configuration for the feed items cleanup scheduler
type FeedItemsCleanupSchedulerConfig struct {
	// CleanupInterval is how often to run the cleanup job (default: daily at 4 AM)
	CleanupInterval time.Duration
	// RunAtStartup determines if cleanup should run immediately on startup
	RunAtStartup bool
}

// DefaultFeedItemsCleanupSchedulerConfig returns default configuration
func DefaultFeedItemsCleanupSchedulerConfig() *FeedItemsCleanupSchedulerConfig {
	return &FeedItemsCleanupSchedulerConfig{
		CleanupInterval: 24 * time.Hour,
		RunAtStartup:    false, // Don't run immediately, wait for first scheduled time
	}
}

// FeedItemsCleanupScheduler manages scheduled execution of the feed items cleanup job
type FeedItemsCleanupScheduler struct {
	config     *FeedItemsCleanupSchedulerConfig
	cleanupJob *FeedItemsCleanupJob
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewFeedItemsCleanupScheduler creates a new feed items cleanup scheduler
func NewFeedItemsCleanupScheduler(
	config *FeedItemsCleanupSchedulerConfig,
	cleanupJob *FeedItemsCleanupJob,
) *FeedItemsCleanupScheduler {
	if config == nil {
		config = DefaultFeedItemsCleanupSchedulerConfig()
	}

	return &FeedItemsCleanupScheduler{
		config:     config,
		cleanupJob: cleanupJob,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the feed items cleanup scheduler
func (s *FeedItemsCleanupScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	log.Printf("Feed items cleanup scheduler started (interval=%v, run_at_startup=%v)",
		s.config.CleanupInterval, s.config.RunAtStartup)
}

// Stop gracefully stops the feed items cleanup scheduler
func (s *FeedItemsCleanupScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("Feed items cleanup scheduler stopped")
}

// run is the main scheduling loop
func (s *FeedItemsCleanupScheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	// Optionally run immediately on start
	if s.config.RunAtStartup {
		s.runCleanup()
	}

	for {
		select {
		case <-ticker.C:
			s.runCleanup()
		case <-s.stopCh:
			return
		}
	}
}

// runCleanup executes the cleanup job with error handling
func (s *FeedItemsCleanupScheduler) runCleanup() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Feed items cleanup job panicked: %v", r)
		}
	}()

	s.cleanupJob.Run()
}

// RunNow triggers an immediate cleanup run (non-blocking)
// This is useful for manual triggers or testing
func (s *FeedItemsCleanupScheduler) RunNow() {
	go s.runCleanup()
}
