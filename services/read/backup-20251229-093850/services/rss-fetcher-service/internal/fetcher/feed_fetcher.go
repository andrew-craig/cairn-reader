package fetcher

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/models"
	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/repository"
	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/scheduler"
	"github.com/google/uuid"
)

// FeedFetcherConfig holds configuration for the feed fetcher
type FeedFetcherConfig struct {
	// Timeout is the HTTP request timeout
	Timeout time.Duration
	// MaxRedirects is the maximum number of redirects to follow
	MaxRedirects int
	// UserAgent is the user agent string to use
	UserAgent string
	// VerifySSL determines whether to verify SSL certificates
	VerifySSL bool
}

// DefaultFeedFetcherConfig returns default configuration
func DefaultFeedFetcherConfig() *FeedFetcherConfig {
	return &FeedFetcherConfig{
		Timeout:      30 * time.Second,
		MaxRedirects: 10,
		UserAgent:    "Cairn-RSS-Fetcher/1.0",
		VerifySSL:    true,
	}
}

// FeedFetcher handles fetching and parsing RSS/Atom feeds
type FeedFetcher struct {
	config       *FeedFetcherConfig
	httpClient   *http.Client
	parser       *Parser
	feedRepo     repository.FeedRepository
	feedItemRepo repository.FeedItemRepository
}

// NewFeedFetcher creates a new feed fetcher
func NewFeedFetcher(
	config *FeedFetcherConfig,
	feedRepo repository.FeedRepository,
	feedItemRepo repository.FeedItemRepository,
) *FeedFetcher {
	if config == nil {
		config = DefaultFeedFetcherConfig()
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.VerifySSL,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		},
	}

	return &FeedFetcher{
		config:       config,
		httpClient:   httpClient,
		parser:       NewParser(),
		feedRepo:     feedRepo,
		feedItemRepo: feedItemRepo,
	}
}

// FetchResult represents the result of fetching a feed
type FetchResult struct {
	Feed         *models.Feed
	NewItems     []*models.FeedItem
	UpdatedItems []*models.FeedItem
	Success      bool
	Error        error
}

// ProcessFeed fetches and processes a feed (implements worker.FeedProcessor)
func (f *FeedFetcher) ProcessFeed(ctx context.Context, feed *models.Feed) error {
	log.Printf("Fetching feed %s (%s)", feed.ID, feed.FeedURL)

	// Fetch and parse the feed
	parsedFeed, err := f.parser.ParseFromURL(ctx, feed.FeedURL, f.httpClient)
	if err != nil {
		return f.handleFetchError(ctx, feed, err)
	}

	// Update feed metadata if available
	if parsedFeed.Title != "" && feed.Title == nil {
		feed.Title = &parsedFeed.Title
	}
	if parsedFeed.Description != "" && feed.Description == nil {
		feed.Description = &parsedFeed.Description
	}
	if parsedFeed.SiteURL != "" && feed.SiteURL == nil {
		feed.SiteURL = &parsedFeed.SiteURL
	}

	// Process feed items
	newItemsCount := 0
	for _, item := range parsedFeed.Items {
		isNew, err := f.processFeedItem(ctx, feed.ID, item)
		if err != nil {
			log.Printf("Error processing feed item %s: %v", item.GUID, err)
			continue
		}
		if isNew {
			newItemsCount++
		}
	}

	// Update feed polling info after successful fetch
	foundNewItems := newItemsCount > 0
	err = scheduler.UpdateFeedPollingAfterSuccess(
		ctx,
		f.feedRepo,
		feed.ID,
		feed.PollingTier,
		foundNewItems,
	)
	if err != nil {
		log.Printf("Error updating feed polling info: %v", err)
	}

	log.Printf("Successfully processed feed %s: %d new items out of %d total items",
		feed.ID, newItemsCount, len(parsedFeed.Items))

	return nil
}

// processFeedItem processes a single feed item
func (f *FeedFetcher) processFeedItem(ctx context.Context, feedID uuid.UUID, item *ParsedItem) (bool, error) {
	// Check if item already exists using feed_id + item_guid
	existingItem, err := f.feedItemRepo.GetByFeedAndGUID(ctx, feedID, item.GUID)
	if err != nil {
		return false, fmt.Errorf("failed to check for existing item: %w", err)
	}

	// If item exists, skip it (we'll handle updates in a separate process)
	if existingItem != nil {
		return false, nil
	}

	// Create new feed item with status 'pending'
	feedItem := &models.FeedItem{
		ID:               uuid.New(),
		FeedID:           feedID,
		ItemURL:          item.URL,
		ItemGUID:         &item.GUID,
		ProcessingStatus: models.ProcessingStatusPending,
		Title:            &item.Title,
		Author:           &item.Author,
		PublishedAt:      item.PublishedAt,
		Description:      &item.Description,
		RetryCount:       0,
		DiscoveredAt:     time.Now(),
	}

	err = f.feedItemRepo.Create(ctx, feedItem)
	if err != nil {
		return false, fmt.Errorf("failed to create feed item: %w", err)
	}

	log.Printf("Created new feed item %s from feed %s", feedItem.ID, feedID)
	return true, nil
}

// handleFetchError handles errors during feed fetching
func (f *FeedFetcher) handleFetchError(ctx context.Context, feed *models.Feed, fetchErr error) error {
	log.Printf("Error fetching feed %s: %v", feed.ID, fetchErr)

	// Increment consecutive error days
	newConsecutiveErrorDays := feed.ConsecutiveErrorDays + 1

	// Update error info
	errorMessage := fetchErr.Error()
	err := scheduler.UpdateFeedPollingAfterError(
		ctx,
		f.feedRepo,
		feed.ID,
		feed.PollingTier,
		newConsecutiveErrorDays,
		errorMessage,
	)
	if err != nil {
		log.Printf("Error updating feed error info: %v", err)
	}

	// Calculate next poll time (still use current tier for retry)
	nextPollAt := scheduler.CalculateNextPollTime(feed.PollingTier)
	now := time.Now()
	err = f.feedRepo.UpdatePollingInfo(ctx, feed.ID, &now, nil, nextPollAt, feed.PollingTier)
	if err != nil {
		log.Printf("Error updating feed next poll time: %v", err)
	}

	return fetchErr
}
