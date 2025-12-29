package processor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/client"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/fetcher"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
	"github.com/go-shiori/go-readability"
	"github.com/microcosm-cc/bluemonday"
)

// UpdateDetectorConfig holds configuration for the update detector
type UpdateDetectorConfig struct {
	// ContentFetchTimeout is the timeout for fetching article content
	ContentFetchTimeout time.Duration
	// MaxContentSize is the maximum content size in bytes (5MB)
	MaxContentSize int64
	// CheckInterval is how often to check for updates (24 hours)
	CheckInterval time.Duration
}

// DefaultUpdateDetectorConfig returns default configuration
func DefaultUpdateDetectorConfig() *UpdateDetectorConfig {
	return &UpdateDetectorConfig{
		ContentFetchTimeout: 30 * time.Second,
		MaxContentSize:      5 * 1024 * 1024, // 5MB
		CheckInterval:       24 * time.Hour,
	}
}

// UpdateDetector checks for content updates using HTTP caching headers
type UpdateDetector struct {
	config              *UpdateDetectorConfig
	httpClient          *http.Client
	conditionalFetcher  *fetcher.ConditionalFetcher
	sanitizer           *bluemonday.Policy
	feedItemRepo        repository.FeedItemRepository
	contentServiceClient *client.ContentServiceClient
}

// NewUpdateDetector creates a new update detector
func NewUpdateDetector(
	config *UpdateDetectorConfig,
	feedItemRepo repository.FeedItemRepository,
	contentServiceClient *client.ContentServiceClient,
) *UpdateDetector {
	if config == nil {
		config = DefaultUpdateDetectorConfig()
	}

	httpClient := &http.Client{
		Timeout: config.ContentFetchTimeout,
	}

	conditionalFetcher := fetcher.NewConditionalFetcher(httpClient)
	sanitizer := bluemonday.UGCPolicy()

	return &UpdateDetector{
		config:              config,
		httpClient:          httpClient,
		conditionalFetcher:  conditionalFetcher,
		sanitizer:           sanitizer,
		feedItemRepo:        feedItemRepo,
		contentServiceClient: contentServiceClient,
	}
}

// CheckForUpdates checks a batch of items for content updates
func (ud *UpdateDetector) CheckForUpdates(ctx context.Context, batchSize int) error {
	// Get items to check for updates
	items, err := ud.feedItemRepo.GetItemsForUpdateCheck(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get items for update check: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	log.Printf("Checking %d items for content updates", len(items))

	for _, item := range items {
		if err := ud.checkItem(ctx, item); err != nil {
			log.Printf("Error checking item %s for updates: %v", item.ID, err)
		}
	}

	return nil
}

// checkItem checks a single item for content updates
func (ud *UpdateDetector) checkItem(ctx context.Context, item *models.FeedItem) error {
	// Perform conditional fetch
	result, err := ud.conditionalFetcher.FetchForItem(ctx, item)
	if err != nil {
		log.Printf("Conditional fetch failed for item %s: %v", item.ID, err)
		// Update last_checked_at even on error
		now := time.Now()
		_ = ud.feedItemRepo.UpdateContentUpdateInfo(ctx, item.ID, item.HTTPLastModified, item.HTTPETag, &now)
		return err
	}

	// Update caching headers and last_checked_at
	now := time.Now()
	if err := ud.feedItemRepo.UpdateContentUpdateInfo(
		ctx, item.ID, result.LastModified, result.ETag, &now,
	); err != nil {
		log.Printf("Failed to update caching headers for item %s: %v", item.ID, err)
	}

	// If not modified, nothing to do
	if result.NotModified {
		log.Printf("Item %s not modified (HTTP 304)", item.ID)
		return nil
	}

	// Content was modified, process the update
	log.Printf("Item %s has been modified, processing update", item.ID)

	// Apply readability parsing
	cleanedHTML, err := ud.applyReadability(item.ItemURL, string(result.Content))
	if err != nil {
		log.Printf("Readability parsing failed for update, using sanitized raw content: %v", err)
		cleanedHTML = string(result.Content)
	}

	// Sanitize HTML
	sanitizedHTML := ud.sanitizer.Sanitize(cleanedHTML)

	// Generate new content hash
	newContentHash := ud.generateContentHash(sanitizedHTML)

	// Compare with stored hash to confirm actual change
	if item.ContentHash != nil && *item.ContentHash == newContentHash {
		log.Printf("Item %s: Headers changed but content hash unchanged, skipping update", item.ID)
		return nil
	}

	// Content has actually changed, update via Content Service
	if item.ContentServiceID == nil {
		log.Printf("Item %s: No content_service_id, skipping update", item.ID)
		return nil
	}

	// Call Content Service to update the content
	updateReq := client.UpdateContentRequest{
		URL:         item.ItemURL,
		HTML:        sanitizedHTML,
		PublishedAt: item.PublishedAt,
	}

	_, err = ud.contentServiceClient.UpdateContent(ctx, *item.ContentServiceID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update content in Content Service: %w", err)
	}

	// Update content_hash in feed_items
	if err := ud.feedItemRepo.UpdateProcessingStatus(
		ctx, item.ID, models.ProcessingStatusCompleted, &newContentHash, item.ContentServiceID, nil,
	); err != nil {
		log.Printf("Failed to update content hash for item %s: %v", item.ID, err)
	}

	log.Printf("Successfully updated content for item %s (content_service_id: %s)", item.ID, item.ContentServiceID)
	return nil
}

// applyReadability applies readability parsing to extract article content
func (ud *UpdateDetector) applyReadability(url, htmlContent string) (string, error) {
	reader := bytes.NewReader([]byte(htmlContent))
	article, err := readability.FromReader(reader, nil)
	if err != nil {
		return "", fmt.Errorf("readability parsing failed: %w", err)
	}

	return article.Content, nil
}

// generateContentHash generates a SHA-256 hash of the content
func (ud *UpdateDetector) generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
