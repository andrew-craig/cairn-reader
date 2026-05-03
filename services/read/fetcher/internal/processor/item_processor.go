package processor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/google/uuid"
)

// ItemProcessorConfig holds configuration for the item processor
type ItemProcessorConfig struct {
	// ContentFetchTimeout is the timeout for fetching article content
	ContentFetchTimeout time.Duration
	// MaxContentSize is the maximum content size in bytes (5MB)
	MaxContentSize int64
	// MaxRetries is the maximum number of retries for failed processing
	MaxRetries int
}

// DefaultItemProcessorConfig returns default configuration
func DefaultItemProcessorConfig() *ItemProcessorConfig {
	return &ItemProcessorConfig{
		ContentFetchTimeout: 30 * time.Second,
		MaxContentSize:      5 * 1024 * 1024, // 5MB
		MaxRetries:          3,
	}
}

// ItemProcessor fetches RSS items and enqueues them, as raw HTML, to the
// content_outbox. Readability + sanitize + hash run exactly once, downstream
// in the Content Service.
type ItemProcessor struct {
	config           *ItemProcessorConfig
	httpClient       *http.Client
	feedItemRepo     repository.FeedItemRepository
	subscriptionRepo repository.FeedSubscriptionRepository
	outboxRepo       repository.OutboxRepository
}

// NewItemProcessor creates a new item processor
func NewItemProcessor(
	config *ItemProcessorConfig,
	feedItemRepo repository.FeedItemRepository,
	subscriptionRepo repository.FeedSubscriptionRepository,
	outboxRepo repository.OutboxRepository,
) *ItemProcessor {
	if config == nil {
		config = DefaultItemProcessorConfig()
	}

	httpClient := &http.Client{
		Timeout: config.ContentFetchTimeout,
	}

	return &ItemProcessor{
		config:           config,
		httpClient:       httpClient,
		feedItemRepo:     feedItemRepo,
		subscriptionRepo: subscriptionRepo,
		outboxRepo:       outboxRepo,
	}
}

// ProcessPendingItems processes a batch of pending feed items
func (p *ItemProcessor) ProcessPendingItems(ctx context.Context, batchSize int) error {
	// Get pending items
	items, err := p.feedItemRepo.GetPendingItems(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	slog.Info("Processing pending feed items", "count", len(items))

	for _, item := range items {
		if err := p.processItem(ctx, item); err != nil {
			slog.Error("Error processing item", "item_id", item.ID, "error", err)

			// Increment retry count
			if item.RetryCount < p.config.MaxRetries {
				if err := p.feedItemRepo.IncrementRetryCount(ctx, item.ID, err.Error()); err != nil {
					slog.Error("Error incrementing retry count", "item_id", item.ID, "error", err)
				}
			} else {
				// Max retries exceeded, mark as failed
				now := time.Now()
				slog.Warn("Max retries exceeded for item", "item_id", item.ID, "error", err)
				if err := p.feedItemRepo.UpdateProcessingStatus(
					ctx, item.ID, models.ProcessingStatusFailed, nil, nil, &now,
				); err != nil {
					slog.Error("Error marking item as failed", "item_id", item.ID, "error", err)
				}
			}
		}
	}

	return nil
}

// processItem processes a single feed item
func (p *ItemProcessor) processItem(ctx context.Context, item *models.FeedItem) error {
	slog.Info("Processing feed item", "item_id", item.ID, "url", item.ItemURL)

	// Update status to 'processing'
	if err := p.feedItemRepo.UpdateProcessingStatus(
		ctx, item.ID, models.ProcessingStatusProcessing, nil, nil, nil,
	); err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	rawHTML, err := p.fetchArticleContent(ctx, item.ItemURL)
	if err != nil {
		slog.Warn("Failed to fetch article content, using RSS description", "error", err)
		if item.Description == nil || *item.Description == "" {
			return fmt.Errorf("failed to fetch article content and no description available: %w", err)
		}
		if int64(len(*item.Description)) > p.config.MaxContentSize {
			return fmt.Errorf("RSS description exceeds maximum content size of %d bytes", p.config.MaxContentSize)
		}
		rawHTML = *item.Description
	}

	contentPayload := map[string]interface{}{
		models.PayloadKeyTitle:          item.Title,
		models.PayloadKeyAuthor:         item.Author,
		models.PayloadKeyPublishedAt:    item.PublishedAt,
		models.PayloadKeySourceURL:      item.ItemURL,
		models.PayloadKeyRawHTML:        rawHTML,
		models.PayloadKeySourceFeedID:   item.FeedID.String(),
		models.PayloadKeyRawDescription: item.Description,
	}

	// Get list of users subscribed to this feed
	subscriptions, err := p.subscriptionRepo.GetByFeedID(ctx, item.FeedID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		slog.Info("No subscriptions found for feed, marking item as completed", "feed_id", item.FeedID)
		now := time.Now()
		return p.feedItemRepo.UpdateProcessingStatus(
			ctx, item.ID, models.ProcessingStatusCompleted, nil, nil, &now,
		)
	}

	// Extract user IDs
	userIDs := make([]uuid.UUID, len(subscriptions))
	for i, sub := range subscriptions {
		userIDs[i] = sub.UserID
	}

	// Create outbox entry
	outbox := &models.ContentOutbox{
		ID:             uuid.New(),
		FeedItemID:     item.ID,
		ContentPayload: contentPayload,
		UserIDs:        userIDs,
		DeliveryStatus: models.DeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     6,
		NextRetryAt:    time.Now(),
		CreatedAt:      time.Now(),
	}

	if err := p.outboxRepo.Create(ctx, outbox); err != nil {
		return fmt.Errorf("failed to create outbox entry: %w", err)
	}

	// content_hash is stamped by outbox_worker from the Content Service
	// response after successful delivery; this is the only short-circuit
	// update_detector has to skip unchanged articles.
	now := time.Now()
	if err := p.feedItemRepo.UpdateProcessingStatus(
		ctx, item.ID, models.ProcessingStatusCompleted, nil, nil, &now,
	); err != nil {
		return fmt.Errorf("failed to update item status to completed: %w", err)
	}

	slog.Info("Successfully processed item", "item_id", item.ID, "user_count", len(userIDs))
	return nil
}

// fetchArticleContent fetches the full article HTML and rejects any response
// strictly larger than MaxContentSize. We read MaxContentSize+1 so the size
// check distinguishes "exactly at the cap" from "truncated by LimitReader",
// avoiding partial HTML reaching the Content Service.
func (p *ItemProcessor) fetchArticleContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Cairn-RSS-Fetcher/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, p.config.MaxContentSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(content)) > p.config.MaxContentSize {
		return "", fmt.Errorf("content size exceeds maximum of %d bytes", p.config.MaxContentSize)
	}

	return string(content), nil
}
