package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/google/uuid"
)

// OutboxWorkerConfig holds configuration for the outbox worker pool
type OutboxWorkerConfig struct {
	// WorkerCount is the number of concurrent workers
	WorkerCount int
	// BatchSize is the number of outbox entries to fetch per poll
	BatchSize int
	// PollInterval is how often to check for pending entries
	PollInterval time.Duration
	// MaxRetries is the maximum number of retries before marking as failed
	MaxRetries int
}

// DefaultOutboxWorkerConfig returns default configuration
func DefaultOutboxWorkerConfig() *OutboxWorkerConfig {
	return &OutboxWorkerConfig{
		WorkerCount:  5,
		BatchSize:    20,
		PollInterval: 10 * time.Second,
		MaxRetries:   6,
	}
}

// OutboxWorker manages a pool of workers for concurrent outbox delivery
type OutboxWorker struct {
	config         *OutboxWorkerConfig
	outboxRepo     repository.OutboxRepository
	feedItemRepo   repository.FeedItemRepository
	contentClient  *client.ContentServiceClient
	outboxQueue    chan *models.ContentOutbox
	stopCh         chan struct{}
	wg             sync.WaitGroup
	pollTickerDone chan struct{}
}

// NewOutboxWorker creates a new outbox worker pool. feedItemRepo may be nil
// only in tests that don't exercise the post-delivery write-back path.
func NewOutboxWorker(
	config *OutboxWorkerConfig,
	outboxRepo repository.OutboxRepository,
	feedItemRepo repository.FeedItemRepository,
	contentClient *client.ContentServiceClient,
) *OutboxWorker {
	if config == nil {
		config = DefaultOutboxWorkerConfig()
	}

	return &OutboxWorker{
		config:         config,
		outboxRepo:     outboxRepo,
		feedItemRepo:   feedItemRepo,
		contentClient:  contentClient,
		outboxQueue:    make(chan *models.ContentOutbox, config.BatchSize*2),
		stopCh:         make(chan struct{}),
		pollTickerDone: make(chan struct{}),
	}
}

// Start begins the worker pool and polling
func (ow *OutboxWorker) Start() {
	// Start worker goroutines
	for i := 0; i < ow.config.WorkerCount; i++ {
		ow.wg.Add(1)
		go ow.worker(i)
	}

	// Start polling goroutine
	go ow.pollPendingEntries()

	slog.Info("Outbox worker pool started",
		"workers", ow.config.WorkerCount,
		"batch_size", ow.config.BatchSize,
		"poll_interval", ow.config.PollInterval)
}

// Stop gracefully stops the worker pool
func (ow *OutboxWorker) Stop() {
	close(ow.stopCh)
	<-ow.pollTickerDone // Wait for poller to stop
	close(ow.outboxQueue)
	ow.wg.Wait()
	slog.Info("Outbox worker pool stopped")
}

// pollPendingEntries continuously polls for pending outbox entries
func (ow *OutboxWorker) pollPendingEntries() {
	defer close(ow.pollTickerDone)

	ticker := time.NewTicker(ow.config.PollInterval)
	defer ticker.Stop()

	// Do an initial poll immediately
	ow.fetchAndQueuePendingEntries()

	for {
		select {
		case <-ticker.C:
			ow.fetchAndQueuePendingEntries()
		case <-ow.stopCh:
			slog.Info("Outbox poller received stop signal")
			return
		}
	}
}

// fetchAndQueuePendingEntries fetches pending entries and queues them for processing
func (ow *OutboxWorker) fetchAndQueuePendingEntries() {
	ctx := context.Background()

	entries, err := ow.outboxRepo.GetPendingEntries(ctx, ow.config.BatchSize)
	if err != nil {
		slog.Error("Error fetching pending outbox entries", "error", err)
		return
	}

	if len(entries) > 0 {
		slog.Info("Fetched pending outbox entries for delivery", "count", len(entries))
	}

	for _, entry := range entries {
		select {
		case ow.outboxQueue <- entry:
			// Entry queued successfully
		case <-ow.stopCh:
			// Worker is stopping
			return
		default:
			// Queue is full, log warning
			slog.Warn("Outbox queue is full, will retry entry on next poll", "entry_id", entry.ID)
		}
	}
}

// worker is a single worker goroutine that processes outbox entries
func (ow *OutboxWorker) worker(id int) {
	defer ow.wg.Done()

	slog.Info("Outbox worker started", "worker_id", id)

	for {
		select {
		case entry, ok := <-ow.outboxQueue:
			if !ok {
				// Channel closed, worker should exit
				slog.Info("Outbox worker stopped", "worker_id", id)
				return
			}
			ow.processOutboxEntry(id, entry)

		case <-ow.stopCh:
			slog.Info("Outbox worker received stop signal", "worker_id", id)
			return
		}
	}
}

// processOutboxEntry processes a single outbox entry
func (ow *OutboxWorker) processOutboxEntry(workerID int, entry *models.ContentOutbox) {
	ctx := context.Background()

	slog.Info("Processing outbox entry",
		"worker_id", workerID,
		"entry_id", entry.ID,
		"retry_count", entry.RetryCount,
		"user_count", len(entry.UserIDs))

	// Set status to 'sending'
	if err := ow.updateStatus(ctx, entry.ID, models.DeliveryStatusSending, nil, nil, nil); err != nil {
		slog.Error("Failed to update status to sending",
			"worker_id", workerID,
			"entry_id", entry.ID,
			"error", err)
		return
	}

	// Extract content data from payload
	contentItem, err := ow.buildContentItem(entry)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to build content item: %v", err)
		slog.Error("Failed to build content item", "worker_id", workerID, "error", err)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	// Call Content Service to create content
	bulkResp, err := ow.contentClient.BulkCreateContent(ctx, []client.BulkContentItem{contentItem})
	if err != nil {
		errorMsg := fmt.Sprintf("Content Service API error: %v", err)
		slog.Error("Content Service API error", "worker_id", workerID, "error", err)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	var contentID uuid.UUID
	var contentHash string
	if len(bulkResp.Created) > 0 {
		contentID = bulkResp.Created[0].ID
		contentHash = bulkResp.Created[0].ContentHash
		slog.Info("Created new content", "worker_id", workerID, "content_id", contentID)
	} else if len(bulkResp.Existing) > 0 {
		contentID = bulkResp.Existing[0].ID
		contentHash = bulkResp.Existing[0].ContentHash
		slog.Info("Using existing content", "worker_id", workerID, "content_id", contentID)
	} else if len(bulkResp.Failed) > 0 {
		errorMsg := fmt.Sprintf("Content creation failed: %s - %s",
			bulkResp.Failed[0].Error, bulkResp.Failed[0].Message)
		slog.Error("Content creation failed", "worker_id", workerID, "error", errorMsg)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	} else {
		errorMsg := "No content created, existing, or failed - unexpected response"
		slog.Error("Unexpected response from Content Service", "worker_id", workerID)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	// Add content to all users' reading lists
	items := make([]client.BulkAddToUsersItem, len(entry.UserIDs))
	for i, userID := range entry.UserIDs {
		items[i] = client.BulkAddToUsersItem{
			ContentID: contentID,
			UserID:    userID,
			Status:    "unread",
		}
	}

	addResp, err := ow.contentClient.AddContentToUsers(ctx, items)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to add content to users: %v", err)
		slog.Error("Failed to add content to users", "worker_id", workerID, "error", err)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	// Check if any additions failed
	if len(addResp.Failed) > 0 {
		slog.Warn("Some users failed to receive content",
			"worker_id", workerID,
			"failed_count", len(addResp.Failed),
			"total_count", len(entry.UserIDs))
		// Still mark as delivered if at least some succeeded
	}

	// Mark as delivered
	deliveredAt := time.Now()
	if err := ow.updateStatus(ctx, entry.ID, models.DeliveryStatusDelivered,
		&contentID, &deliveredAt, nil); err != nil {
		slog.Error("Failed to update status to delivered",
			"worker_id", workerID,
			"entry_id", entry.ID,
			"error", err)
		return
	}

	// Best-effort write-back. A failure here doesn't fail the delivery (the
	// content is already in the user's reading list), but it does cost
	// update_detector its no-op short-circuit for this article.
	if ow.feedItemRepo != nil {
		var hashPtr *string
		if contentHash != "" {
			hashPtr = &contentHash
		}
		if err := ow.feedItemRepo.UpdateProcessingStatus(
			ctx, entry.FeedItemID, models.ProcessingStatusCompleted,
			hashPtr, &contentID, &deliveredAt,
		); err != nil {
			slog.Warn("Failed to stamp content_hash/content_service_id on feed_item",
				"worker_id", workerID,
				"feed_item_id", entry.FeedItemID,
				"error", err)
		}
	}

	slog.Info("Successfully delivered outbox entry",
		"worker_id", workerID,
		"entry_id", entry.ID,
		"user_count", len(addResp.Succeeded),
		"content_id", contentID)
}

// buildContentItem builds a BulkContentItem from the outbox entry's payload.
func (ow *OutboxWorker) buildContentItem(entry *models.ContentOutbox) (client.BulkContentItem, error) {
	payload := entry.ContentPayload

	url, ok := payload[models.PayloadKeySourceURL].(string)
	if !ok || url == "" {
		return client.BulkContentItem{}, fmt.Errorf("missing or invalid '%s' field", models.PayloadKeySourceURL)
	}

	html, _ := payload[models.PayloadKeyRawHTML].(string)
	if html == "" {
		return client.BulkContentItem{}, fmt.Errorf("missing or invalid '%s' field", models.PayloadKeyRawHTML)
	}

	item := client.BulkContentItem{
		URL:        url,
		HTML:       html,
		SourceType: "rss",
	}

	if feedIDStr, ok := payload[models.PayloadKeySourceFeedID].(string); ok && feedIDStr != "" {
		if feedID, err := uuid.Parse(feedIDStr); err == nil {
			item.SourceFeedID = &feedID
		}
	}

	if publishedAtStr, ok := payload[models.PayloadKeyPublishedAt].(string); ok && publishedAtStr != "" {
		if publishedAt, err := time.Parse(time.RFC3339, publishedAtStr); err == nil {
			item.PublishedAt = &publishedAt
		}
	}

	if title, ok := payload[models.PayloadKeyTitle].(string); ok && title != "" {
		item.Title = &title
	}

	if author, ok := payload[models.PayloadKeyAuthor].(string); ok && author != "" {
		item.Author = &author
	}

	return item, nil
}

// handleDeliveryFailure handles a failed delivery attempt
func (ow *OutboxWorker) handleDeliveryFailure(ctx context.Context, entry *models.ContentOutbox, errorMsg string) {
	// Increment retry count
	nextRetryAt := ow.calculateNextRetry(entry.RetryCount + 1)

	if err := ow.outboxRepo.IncrementRetryCount(ctx, entry.ID, nextRetryAt, errorMsg); err != nil {
		slog.Error("Failed to increment retry count", "entry_id", entry.ID, "error", err)
		return
	}

	// Check if max retries exceeded (IncrementRetryCount will set status to 'failed' if so)
	if entry.RetryCount+1 >= ow.config.MaxRetries {
		slog.Warn("Outbox entry marked as failed after max retries",
			"entry_id", entry.ID,
			"retry_count", entry.RetryCount+1)
	} else {
		slog.Info("Outbox entry will retry",
			"entry_id", entry.ID,
			"retry_in", time.Until(nextRetryAt).Round(time.Second),
			"retry_number", entry.RetryCount+1,
			"max_retries", ow.config.MaxRetries)
	}
}

// calculateNextRetry calculates the next retry time using exponential backoff
// Retry schedule:
// - Retry 1: 1 minute
// - Retry 2: 5 minutes
// - Retry 3: 15 minutes
// - Retry 4: 1 hour
// - Retry 5: 4 hours
// - Retry 6: 12 hours
func (ow *OutboxWorker) calculateNextRetry(retryCount int) time.Time {
	var delay time.Duration

	switch retryCount {
	case 1:
		delay = 1 * time.Minute
	case 2:
		delay = 5 * time.Minute
	case 3:
		delay = 15 * time.Minute
	case 4:
		delay = 1 * time.Hour
	case 5:
		delay = 4 * time.Hour
	case 6:
		delay = 12 * time.Hour
	default:
		// Should not reach here, but default to 1 hour
		delay = 1 * time.Hour
	}

	return time.Now().Add(delay)
}

// updateStatus updates the delivery status of an outbox entry
func (ow *OutboxWorker) updateStatus(
	ctx context.Context,
	id uuid.UUID,
	status models.DeliveryStatus,
	contentServiceID *uuid.UUID,
	deliveredAt *time.Time,
	lastError *string,
) error {
	return ow.outboxRepo.UpdateDeliveryStatus(ctx, id, status, contentServiceID, deliveredAt, lastError)
}

// QueueSize returns the current number of entries in the queue
func (ow *OutboxWorker) QueueSize() int {
	return len(ow.outboxQueue)
}
