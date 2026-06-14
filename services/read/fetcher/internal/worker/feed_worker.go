package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
)

// FeedWorkerConfig holds configuration for the feed worker pool
type FeedWorkerConfig struct {
	// WorkerCount is the number of concurrent workers
	WorkerCount int
	// QueueSize is the size of the feed queue
	QueueSize int
}

// DefaultFeedWorkerConfig returns default configuration
func DefaultFeedWorkerConfig() *FeedWorkerConfig {
	return &FeedWorkerConfig{
		WorkerCount: 5,
		QueueSize:   100,
	}
}

// FeedProcessor defines the interface for processing a feed
type FeedProcessor interface {
	ProcessFeed(ctx context.Context, feed *models.Feed) error
}

// FeedWorker manages a pool of workers for concurrent feed processing
type FeedWorker struct {
	config    *FeedWorkerConfig
	feedRepo  repository.FeedRepository
	processor FeedProcessor
	feedQueue chan *models.Feed
	stopCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	stopped   bool
}

// NewFeedWorker creates a new feed worker pool
func NewFeedWorker(
	config *FeedWorkerConfig,
	feedRepo repository.FeedRepository,
	processor FeedProcessor,
) *FeedWorker {
	if config == nil {
		config = DefaultFeedWorkerConfig()
	}

	return &FeedWorker{
		config:    config,
		feedRepo:  feedRepo,
		processor: processor,
		feedQueue: make(chan *models.Feed, config.QueueSize),
		stopCh:    make(chan struct{}),
	}
}

// Start begins the worker pool
func (fw *FeedWorker) Start() {
	for i := 0; i < fw.config.WorkerCount; i++ {
		fw.wg.Add(1)
		go fw.worker(i)
	}
	slog.Info("Feed worker pool started",
		"workers", fw.config.WorkerCount,
		"queue_size", fw.config.QueueSize)
}

// Stop gracefully stops the worker pool
func (fw *FeedWorker) Stop() {
	fw.mu.Lock()
	if fw.stopped {
		fw.mu.Unlock()
		return
	}
	fw.stopped = true
	close(fw.feedQueue)
	fw.mu.Unlock()
	fw.wg.Wait()
	slog.Info("Feed worker pool stopped")
}

// Submit submits a feed for processing
func (fw *FeedWorker) Submit(feed *models.Feed) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.stopped {
		slog.Warn("Worker pool is stopping, skipping feed", "feed_id", feed.ID)
		return
	}

	select {
	case fw.feedQueue <- feed:
		// Feed submitted successfully
	default:
		// Queue is full, log warning
		slog.Warn("Feed queue is full, skipping feed", "feed_id", feed.ID)
	}
}

// worker is a single worker goroutine that processes feeds
func (fw *FeedWorker) worker(id int) {
	defer fw.wg.Done()

	slog.Info("Feed worker started", "worker_id", id)

	for {
		select {
		case feed, ok := <-fw.feedQueue:
			if !ok {
				// Channel closed, worker should exit
				slog.Info("Feed worker stopped", "worker_id", id)
				return
			}
			fw.processFeed(id, feed)

		case <-fw.stopCh:
			slog.Info("Feed worker received stop signal", "worker_id", id)
			return
		}
	}
}

// processFeed processes a single feed
func (fw *FeedWorker) processFeed(workerID int, feed *models.Feed) {
	slog.Info("Processing feed", "worker_id", workerID, "feed_id", feed.ID, "feed_url", feed.FeedURL)

	ctx := context.Background()

	// Use the processor if provided, otherwise just log
	if fw.processor != nil {
		if err := fw.processor.ProcessFeed(ctx, feed); err != nil {
			slog.Error("Error processing feed", "worker_id", workerID, "feed_id", feed.ID, "error", err)
		} else {
			slog.Info("Successfully processed feed", "worker_id", workerID, "feed_id", feed.ID)
		}
	} else {
		// Placeholder: In Phase 3.2, this will fetch and parse the feed
		slog.Debug("Feed processor not implemented yet (Phase 3.2)", "worker_id", workerID)
		// For now, we'll just log that we received the feed
		// The actual RSS fetching and parsing will be implemented in Phase 3.2
	}
}

// QueueSize returns the current number of feeds in the queue
func (fw *FeedWorker) QueueSize() int {
	return len(fw.feedQueue)
}

// NoOpFeedProcessor is a no-op processor for testing/placeholder
type NoOpFeedProcessor struct{}

// ProcessFeed does nothing (placeholder implementation)
func (n *NoOpFeedProcessor) ProcessFeed(ctx context.Context, feed *models.Feed) error {
	slog.Debug("NoOpFeedProcessor would process feed", "feed_id", feed.ID, "feed_url", feed.FeedURL)
	return nil
}
