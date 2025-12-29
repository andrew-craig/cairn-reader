package processor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/models"
	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/repository"
	"github.com/go-shiori/go-readability"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
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

// ItemProcessor processes pending feed items
type ItemProcessor struct {
	config           *ItemProcessorConfig
	httpClient       *http.Client
	sanitizer        *bluemonday.Policy
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

	// Create sanitizer policy (allows safe HTML tags)
	sanitizer := bluemonday.UGCPolicy()

	return &ItemProcessor{
		config:           config,
		httpClient:       httpClient,
		sanitizer:        sanitizer,
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

	log.Printf("Processing %d pending feed items", len(items))

	for _, item := range items {
		if err := p.processItem(ctx, item); err != nil {
			log.Printf("Error processing item %s: %v", item.ID, err)

			// Increment retry count
			if item.RetryCount < p.config.MaxRetries {
				if err := p.feedItemRepo.IncrementRetryCount(ctx, item.ID, err.Error()); err != nil {
					log.Printf("Error incrementing retry count for item %s: %v", item.ID, err)
				}
			} else {
				// Max retries exceeded, mark as failed
				now := time.Now()
				log.Printf("Max retries exceeded for item %s: %v", item.ID, err)
				if err := p.feedItemRepo.UpdateProcessingStatus(
					ctx, item.ID, models.ProcessingStatusFailed, nil, nil, &now,
				); err != nil {
					log.Printf("Error marking item %s as failed: %v", item.ID, err)
				}
			}
		}
	}

	return nil
}

// processItem processes a single feed item
func (p *ItemProcessor) processItem(ctx context.Context, item *models.FeedItem) error {
	log.Printf("Processing feed item %s (%s)", item.ID, item.ItemURL)

	// Update status to 'processing'
	if err := p.feedItemRepo.UpdateProcessingStatus(
		ctx, item.ID, models.ProcessingStatusProcessing, nil, nil, nil,
	); err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	// Fetch article content
	content, err := p.fetchArticleContent(ctx, item.ItemURL)
	if err != nil {
		// Fallback to RSS description if content fetch fails
		log.Printf("Failed to fetch article content, using RSS description: %v", err)
		if item.Description != nil && *item.Description != "" {
			content = *item.Description
		} else {
			return fmt.Errorf("failed to fetch article content and no description available: %w", err)
		}
	}

	// Apply readability parsing
	cleanedHTML, err := p.applyReadability(item.ItemURL, content)
	if err != nil {
		log.Printf("Readability parsing failed, using sanitized raw content: %v", err)
		cleanedHTML = content
	}

	// Sanitize HTML
	sanitizedHTML := p.sanitizer.Sanitize(cleanedHTML)

	// Generate content hash
	contentHash := p.generateContentHash(sanitizedHTML)

	// Create content payload for Content Service
	contentPayload := map[string]interface{}{
		"title":           item.Title,
		"author":          item.Author,
		"published_at":    item.PublishedAt,
		"source_url":      item.ItemURL,
		"cleaned_html":    sanitizedHTML,
		"content_hash":    contentHash,
		"source_feed_id":  item.FeedID.String(),
		"raw_description": item.Description,
	}

	// Get list of users subscribed to this feed
	subscriptions, err := p.subscriptionRepo.GetByFeedID(ctx, item.FeedID)
	if err != nil {
		return fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if len(subscriptions) == 0 {
		log.Printf("No subscriptions found for feed %s, marking item as completed", item.FeedID)
		now := time.Now()
		return p.feedItemRepo.UpdateProcessingStatus(
			ctx, item.ID, models.ProcessingStatusCompleted, &contentHash, nil, &now,
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

	// Update item status to 'completed'
	now := time.Now()
	if err := p.feedItemRepo.UpdateProcessingStatus(
		ctx, item.ID, models.ProcessingStatusCompleted, &contentHash, nil, &now,
	); err != nil {
		return fmt.Errorf("failed to update item status to completed: %w", err)
	}

	log.Printf("Successfully processed item %s, created outbox entry for %d users", item.ID, len(userIDs))
	return nil
}

// fetchArticleContent fetches the full article content from the source URL
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response with size limit
	limitedReader := io.LimitReader(resp.Body, p.config.MaxContentSize)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(content), nil
}

// applyReadability applies readability parsing to extract article content
func (p *ItemProcessor) applyReadability(url, htmlContent string) (string, error) {
	reader := bytes.NewReader([]byte(htmlContent))
	article, err := readability.FromReader(reader, nil)
	if err != nil {
		return "", fmt.Errorf("readability parsing failed: %w", err)
	}

	return article.Content, nil
}

// generateContentHash generates a SHA-256 hash of the content
func (p *ItemProcessor) generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
