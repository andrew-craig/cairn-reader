package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/client"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
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
	contentClient  *client.ContentServiceClient
	outboxQueue    chan *models.ContentOutbox
	stopCh         chan struct{}
	wg             sync.WaitGroup
	pollTickerDone chan struct{}
}

// NewOutboxWorker creates a new outbox worker pool
func NewOutboxWorker(
	config *OutboxWorkerConfig,
	outboxRepo repository.OutboxRepository,
	contentClient *client.ContentServiceClient,
) *OutboxWorker {
	if config == nil {
		config = DefaultOutboxWorkerConfig()
	}

	return &OutboxWorker{
		config:         config,
		outboxRepo:     outboxRepo,
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

	log.Printf("Outbox worker pool started with %d workers (batch_size=%d, poll_interval=%s)",
		ow.config.WorkerCount, ow.config.BatchSize, ow.config.PollInterval)
}

// Stop gracefully stops the worker pool
func (ow *OutboxWorker) Stop() {
	close(ow.stopCh)
	<-ow.pollTickerDone // Wait for poller to stop
	close(ow.outboxQueue)
	ow.wg.Wait()
	log.Println("Outbox worker pool stopped")
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
			log.Println("Outbox poller received stop signal")
			return
		}
	}
}

// fetchAndQueuePendingEntries fetches pending entries and queues them for processing
func (ow *OutboxWorker) fetchAndQueuePendingEntries() {
	ctx := context.Background()

	entries, err := ow.outboxRepo.GetPendingEntries(ctx, ow.config.BatchSize)
	if err != nil {
		log.Printf("Error fetching pending outbox entries: %v", err)
		return
	}

	if len(entries) > 0 {
		log.Printf("Fetched %d pending outbox entries for delivery", len(entries))
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
			log.Printf("Outbox queue is full, will retry entry %s on next poll", entry.ID)
		}
	}
}

// worker is a single worker goroutine that processes outbox entries
func (ow *OutboxWorker) worker(id int) {
	defer ow.wg.Done()

	log.Printf("Outbox worker %d started", id)

	for {
		select {
		case entry, ok := <-ow.outboxQueue:
			if !ok {
				// Channel closed, worker should exit
				log.Printf("Outbox worker %d stopped", id)
				return
			}
			ow.processOutboxEntry(id, entry)

		case <-ow.stopCh:
			log.Printf("Outbox worker %d received stop signal", id)
			return
		}
	}
}

// processOutboxEntry processes a single outbox entry
func (ow *OutboxWorker) processOutboxEntry(workerID int, entry *models.ContentOutbox) {
	ctx := context.Background()

	log.Printf("Worker %d processing outbox entry %s (retry_count=%d, users=%d)",
		workerID, entry.ID, entry.RetryCount, len(entry.UserIDs))

	// Set status to 'sending'
	if err := ow.updateStatus(ctx, entry.ID, models.DeliveryStatusSending, nil, nil, nil); err != nil {
		log.Printf("Worker %d failed to update status to sending for entry %s: %v",
			workerID, entry.ID, err)
		return
	}

	// Extract content data from payload
	contentItem, err := ow.buildContentItem(entry)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to build content item: %v", err)
		log.Printf("Worker %d: %s", workerID, errorMsg)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	// Call Content Service to create content
	bulkResp, err := ow.contentClient.BulkCreateContent(ctx, []client.BulkContentItem{contentItem})
	if err != nil {
		errorMsg := fmt.Sprintf("Content Service API error: %v", err)
		log.Printf("Worker %d: %s", workerID, errorMsg)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	// Determine which content to use (created or existing)
	var contentID uuid.UUID
	if len(bulkResp.Created) > 0 {
		contentID = bulkResp.Created[0].ID
		log.Printf("Worker %d: Created new content with ID %s", workerID, contentID)
	} else if len(bulkResp.Existing) > 0 {
		contentID = bulkResp.Existing[0].ID
		log.Printf("Worker %d: Using existing content with ID %s", workerID, contentID)
	} else if len(bulkResp.Failed) > 0 {
		errorMsg := fmt.Sprintf("Content creation failed: %s - %s",
			bulkResp.Failed[0].Error, bulkResp.Failed[0].Message)
		log.Printf("Worker %d: %s", workerID, errorMsg)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	} else {
		errorMsg := "No content created, existing, or failed - unexpected response"
		log.Printf("Worker %d: %s", workerID, errorMsg)
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
		log.Printf("Worker %d: %s", workerID, errorMsg)
		ow.handleDeliveryFailure(ctx, entry, errorMsg)
		return
	}

	// Check if any additions failed
	if len(addResp.Failed) > 0 {
		log.Printf("Worker %d: %d/%d users failed to receive content",
			workerID, len(addResp.Failed), len(entry.UserIDs))
		// Still mark as delivered if at least some succeeded
	}

	// Mark as delivered
	deliveredAt := time.Now()
	if err := ow.updateStatus(ctx, entry.ID, models.DeliveryStatusDelivered,
		&contentID, &deliveredAt, nil); err != nil {
		log.Printf("Worker %d: Failed to update status to delivered for entry %s: %v",
			workerID, entry.ID, err)
		return
	}

	log.Printf("Worker %d: Successfully delivered outbox entry %s to %d users (content_id=%s)",
		workerID, entry.ID, len(addResp.Succeeded), contentID)
}

// buildContentItem builds a BulkContentItem from the outbox entry's payload
func (ow *OutboxWorker) buildContentItem(entry *models.ContentOutbox) (client.BulkContentItem, error) {
	payload := entry.ContentPayload

	// Extract required fields
	url, ok := payload["url"].(string)
	if !ok || url == "" {
		return client.BulkContentItem{}, fmt.Errorf("missing or invalid 'url' field")
	}

	html, ok := payload["html"].(string)
	if !ok || html == "" {
		return client.BulkContentItem{}, fmt.Errorf("missing or invalid 'html' field")
	}

	item := client.BulkContentItem{
		URL:        url,
		HTML:       html,
		SourceType: "rss",
	}

	// Extract optional feed ID
	if feedIDStr, ok := payload["source_feed_id"].(string); ok && feedIDStr != "" {
		feedID, err := uuid.Parse(feedIDStr)
		if err == nil {
			item.SourceFeedID = &feedID
		}
	}

	// Extract optional published_at
	if publishedAtStr, ok := payload["published_at"].(string); ok && publishedAtStr != "" {
		publishedAt, err := time.Parse(time.RFC3339, publishedAtStr)
		if err == nil {
			item.PublishedAt = &publishedAt
		}
	}

	return item, nil
}

// handleDeliveryFailure handles a failed delivery attempt
func (ow *OutboxWorker) handleDeliveryFailure(ctx context.Context, entry *models.ContentOutbox, errorMsg string) {
	// Increment retry count
	nextRetryAt := ow.calculateNextRetry(entry.RetryCount + 1)

	if err := ow.outboxRepo.IncrementRetryCount(ctx, entry.ID, nextRetryAt, errorMsg); err != nil {
		log.Printf("Failed to increment retry count for entry %s: %v", entry.ID, err)
		return
	}

	// Check if max retries exceeded (IncrementRetryCount will set status to 'failed' if so)
	if entry.RetryCount+1 >= ow.config.MaxRetries {
		log.Printf("Outbox entry %s marked as failed after %d retries", entry.ID, entry.RetryCount+1)
	} else {
		log.Printf("Outbox entry %s will retry in %s (retry %d/%d)",
			entry.ID, time.Until(nextRetryAt).Round(time.Second), entry.RetryCount+1, ow.config.MaxRetries)
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
