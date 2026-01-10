package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/processor"
)

// ContentExtractionJobConfig holds configuration for the content extraction job
type ContentExtractionJobConfig struct {
	// BatchSize is the number of feed items to process per batch
	BatchSize int
	// ProcessInterval is how often to check for pending items
	ProcessInterval time.Duration
}

// DefaultContentExtractionJobConfig returns default configuration
func DefaultContentExtractionJobConfig() *ContentExtractionJobConfig {
	return &ContentExtractionJobConfig{
		BatchSize:       20,
		ProcessInterval: 30 * time.Second,
	}
}

// ContentExtractionJob manages the processing of pending feed items
type ContentExtractionJob struct {
	config        *ContentExtractionJobConfig
	itemProcessor *processor.ItemProcessor
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewContentExtractionJob creates a new content extraction job
func NewContentExtractionJob(
	config *ContentExtractionJobConfig,
	itemProcessor *processor.ItemProcessor,
) *ContentExtractionJob {
	if config == nil {
		config = DefaultContentExtractionJobConfig()
	}

	return &ContentExtractionJob{
		config:        config,
		itemProcessor: itemProcessor,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the content extraction job
func (j *ContentExtractionJob) Start() {
	j.wg.Add(1)
	go j.run()
	slog.Info("Content extraction job started",
		"batch_size", j.config.BatchSize,
		"interval", j.config.ProcessInterval)
}

// Stop gracefully stops the content extraction job
func (j *ContentExtractionJob) Stop() {
	close(j.stopCh)
	j.wg.Wait()
	slog.Info("Content extraction job stopped")
}

// run is the main processing loop
func (j *ContentExtractionJob) run() {
	defer j.wg.Done()

	ticker := time.NewTicker(j.config.ProcessInterval)
	defer ticker.Stop()

	// Run immediately on start
	j.processPendingItems()

	for {
		select {
		case <-ticker.C:
			j.processPendingItems()
		case <-j.stopCh:
			return
		}
	}
}

// processPendingItems processes a batch of pending feed items
func (j *ContentExtractionJob) processPendingItems() {
	ctx := context.Background()

	if err := j.itemProcessor.ProcessPendingItems(ctx, j.config.BatchSize); err != nil {
		slog.Error("Error processing pending items", "error", err)
	}
}
