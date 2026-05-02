package processor

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/fetcher"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
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
		MaxContentSize:      5 * 1024 * 1024,
		CheckInterval:       24 * time.Hour,
	}
}

// UpdateDetector checks for content updates using HTTP caching headers and
// re-publishes raw HTML to the Content Service when an article changes. The
// Content Service is the single place that runs readability + sanitize +
// content-hash; this detector deliberately does none of that work locally.
type UpdateDetector struct {
	config               *UpdateDetectorConfig
	conditionalFetcher   *fetcher.ConditionalFetcher
	feedItemRepo         repository.FeedItemRepository
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

	httpClient := &http.Client{Timeout: config.ContentFetchTimeout}
	conditionalFetcher := fetcher.NewConditionalFetcher(httpClient)

	return &UpdateDetector{
		config:               config,
		conditionalFetcher:   conditionalFetcher,
		feedItemRepo:         feedItemRepo,
		contentServiceClient: contentServiceClient,
	}
}

// CheckForUpdates checks a batch of items for content updates
func (ud *UpdateDetector) CheckForUpdates(ctx context.Context, batchSize int) error {
	items, err := ud.feedItemRepo.GetItemsForUpdateCheck(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get items for update check: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	slog.Info("Checking items for content updates", "count", len(items))

	for _, item := range items {
		if err := ud.checkItem(ctx, item); err != nil {
			slog.Error("Error checking item for updates", "item_id", item.ID, "error", err)
		}
	}

	return nil
}

// checkItem checks a single item for content updates
func (ud *UpdateDetector) checkItem(ctx context.Context, item *models.FeedItem) error {
	result, err := ud.conditionalFetcher.FetchForItem(ctx, item)
	if err != nil {
		slog.Error("Conditional fetch failed for item", "item_id", item.ID, "error", err)
		now := time.Now()
		_ = ud.feedItemRepo.UpdateContentUpdateInfo(ctx, item.ID, item.HTTPLastModified, item.HTTPETag, &now)
		return err
	}

	now := time.Now()
	if err := ud.feedItemRepo.UpdateContentUpdateInfo(
		ctx, item.ID, result.LastModified, result.ETag, &now,
	); err != nil {
		slog.Error("Failed to update caching headers for item", "item_id", item.ID, "error", err)
	}

	if result.NotModified {
		slog.Debug("Item not modified (HTTP 304)", "item_id", item.ID)
		return nil
	}

	if item.ContentServiceID == nil {
		slog.Debug("No content_service_id, skipping update", "item_id", item.ID)
		return nil
	}

	slog.Info("Item has been modified, processing update", "item_id", item.ID)

	updateReq := client.UpdateContentRequest{
		URL:         item.ItemURL,
		HTML:        string(result.Content),
		PublishedAt: item.PublishedAt,
	}

	resp, err := ud.contentServiceClient.UpdateContent(ctx, *item.ContentServiceID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update content in Content Service: %w", err)
	}

	// Stamp the canonical hash from the Content Service back onto feed_items
	// so future polls can short-circuit unchanged articles via the (now
	// downstream-computed) hash.
	var hashPtr *string
	if resp != nil && resp.ContentHash != "" {
		hashPtr = &resp.ContentHash
	}
	if err := ud.feedItemRepo.UpdateProcessingStatus(
		ctx, item.ID, models.ProcessingStatusCompleted, hashPtr, item.ContentServiceID, nil,
	); err != nil {
		slog.Error("Failed to update content hash for item", "item_id", item.ID, "error", err)
	}

	slog.Info("Successfully updated content for item", "item_id", item.ID, "content_service_id", item.ContentServiceID)
	return nil
}
