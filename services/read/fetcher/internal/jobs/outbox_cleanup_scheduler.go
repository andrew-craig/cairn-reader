package jobs

import (
	"log/slog"
	"sync"
	"time"
)

// OutboxCleanupSchedulerConfig holds configuration for the outbox cleanup scheduler
type OutboxCleanupSchedulerConfig struct {
	// CleanupInterval is how often to run the cleanup job (default: daily at 3 AM)
	CleanupInterval time.Duration
	// RunAtStartup determines if cleanup should run immediately on startup
	RunAtStartup bool
}

// DefaultOutboxCleanupSchedulerConfig returns default configuration
func DefaultOutboxCleanupSchedulerConfig() *OutboxCleanupSchedulerConfig {
	return &OutboxCleanupSchedulerConfig{
		CleanupInterval: 24 * time.Hour,
		RunAtStartup:    false, // Don't run immediately, wait for first scheduled time
	}
}

// OutboxCleanupScheduler manages scheduled execution of the outbox cleanup job
type OutboxCleanupScheduler struct {
	config     *OutboxCleanupSchedulerConfig
	cleanupJob *OutboxCleanupJob
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewOutboxCleanupScheduler creates a new outbox cleanup scheduler
func NewOutboxCleanupScheduler(
	config *OutboxCleanupSchedulerConfig,
	cleanupJob *OutboxCleanupJob,
) *OutboxCleanupScheduler {
	if config == nil {
		config = DefaultOutboxCleanupSchedulerConfig()
	}

	return &OutboxCleanupScheduler{
		config:     config,
		cleanupJob: cleanupJob,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the outbox cleanup scheduler
func (s *OutboxCleanupScheduler) Start() {
	s.wg.Add(1)
	go s.run()
	slog.Info("Outbox cleanup scheduler started",
		"interval", s.config.CleanupInterval,
		"run_at_startup", s.config.RunAtStartup)
}

// Stop gracefully stops the outbox cleanup scheduler
func (s *OutboxCleanupScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	slog.Info("Outbox cleanup scheduler stopped")
}

// run is the main scheduling loop
func (s *OutboxCleanupScheduler) run() {
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
func (s *OutboxCleanupScheduler) runCleanup() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Outbox cleanup job panicked", "panic", r)
		}
	}()

	s.cleanupJob.Run()
}

// RunNow triggers an immediate cleanup run (non-blocking)
// This is useful for manual triggers or testing
func (s *OutboxCleanupScheduler) RunNow() {
	go s.runCleanup()
}
