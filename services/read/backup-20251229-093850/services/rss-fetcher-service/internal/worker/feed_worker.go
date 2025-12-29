package worker

import (
	"context"
	"log"
	"sync"

	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/models"
	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/repository"
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
	log.Printf("Feed worker pool started with %d workers (queue_size=%d)",
		fw.config.WorkerCount, fw.config.QueueSize)
}

// Stop gracefully stops the worker pool
func (fw *FeedWorker) Stop() {
	close(fw.stopCh)
	// Close the feed queue to signal workers to finish
	close(fw.feedQueue)
	fw.wg.Wait()
	log.Println("Feed worker pool stopped")
}

// Submit submits a feed for processing
func (fw *FeedWorker) Submit(feed *models.Feed) {
	select {
	case fw.feedQueue <- feed:
		// Feed submitted successfully
	case <-fw.stopCh:
		// Worker is stopping, don't block
		log.Printf("Worker pool is stopping, skipping feed %s", feed.ID)
	default:
		// Queue is full, log warning
		log.Printf("Feed queue is full, skipping feed %s", feed.ID)
	}
}

// worker is a single worker goroutine that processes feeds
func (fw *FeedWorker) worker(id int) {
	defer fw.wg.Done()

	log.Printf("Worker %d started", id)

	for {
		select {
		case feed, ok := <-fw.feedQueue:
			if !ok {
				// Channel closed, worker should exit
				log.Printf("Worker %d stopped", id)
				return
			}
			fw.processFeed(id, feed)

		case <-fw.stopCh:
			log.Printf("Worker %d received stop signal", id)
			return
		}
	}
}

// processFeed processes a single feed
func (fw *FeedWorker) processFeed(workerID int, feed *models.Feed) {
	log.Printf("Worker %d processing feed %s (%s)", workerID, feed.ID, feed.FeedURL)

	ctx := context.Background()

	// Use the processor if provided, otherwise just log
	if fw.processor != nil {
		if err := fw.processor.ProcessFeed(ctx, feed); err != nil {
			log.Printf("Worker %d error processing feed %s: %v", workerID, feed.ID, err)
		} else {
			log.Printf("Worker %d successfully processed feed %s", workerID, feed.ID)
		}
	} else {
		// Placeholder: In Phase 3.2, this will fetch and parse the feed
		log.Printf("Worker %d: Feed processor not implemented yet (Phase 3.2)", workerID)
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
	log.Printf("NoOpFeedProcessor: Would process feed %s (%s)", feed.ID, feed.FeedURL)
	return nil
}
