package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/rss/fetch"
	"github.com/cairn-app/cairn-reader/pkg/rss/parse"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/google/uuid"
)

const (
	// MaxFeedsPerUser is the maximum number of feeds a user can subscribe to
	MaxFeedsPerUser = 100

	// FeedFetchTimeout is the timeout for fetching a feed
	FeedFetchTimeout = 30 * time.Second
)

// FeedService defines the interface for feed-related business logic
type FeedService interface {
	// Subscribe subscribes a user to a feed
	Subscribe(ctx context.Context, userID uuid.UUID, feedURL string) (*SubscriptionResult, error)

	// Unsubscribe unsubscribes a user from a feed
	Unsubscribe(ctx context.Context, userID, feedID uuid.UUID) error

	// ListUserSubscriptions lists all feeds a user is subscribed to
	ListUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*UserFeedSubscription, error)

	// EnableFeed re-enables a disabled feed
	EnableFeed(ctx context.Context, feedID uuid.UUID) error

	// ValidateAndExtractFeedMetadata validates a feed URL and extracts metadata
	ValidateAndExtractFeedMetadata(ctx context.Context, feedURL string) (*FeedMetadata, error)
}

// SubscriptionResult represents the result of a subscription operation
type SubscriptionResult struct {
	Subscription *models.FeedSubscription `json:"subscription"`
	Feed         *models.Feed             `json:"feed"`
	IsNewFeed    bool                     `json:"is_new_feed"`
}

// UserFeedSubscription represents a user's subscription with feed details
type UserFeedSubscription struct {
	Subscription *models.FeedSubscription `json:"subscription"`
	Feed         *models.Feed             `json:"feed"`
}

// FeedMetadata represents extracted metadata from a feed
type FeedMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	SiteURL     string `json:"site_url"`
}

// feedService is the concrete implementation
type feedService struct {
	feedRepo         repository.FeedRepository
	subscriptionRepo repository.FeedSubscriptionRepository
	httpClient       *http.Client
}

// NewFeedService creates a new FeedService
func NewFeedService(
	feedRepo repository.FeedRepository,
	subscriptionRepo repository.FeedSubscriptionRepository,
) FeedService {
	return &feedService{
		feedRepo:         feedRepo,
		subscriptionRepo: subscriptionRepo,
		httpClient: &http.Client{
			Timeout: FeedFetchTimeout,
		},
	}
}

// Subscribe subscribes a user to a feed
func (s *feedService) Subscribe(ctx context.Context, userID uuid.UUID, feedURL string) (*SubscriptionResult, error) {
	// Check if user has reached the feed limit
	count, err := s.subscriptionRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user subscriptions: %w", err)
	}

	if count >= MaxFeedsPerUser {
		return nil, fmt.Errorf("user has reached maximum feed limit of %d", MaxFeedsPerUser)
	}

	// Check if feed already exists
	existingFeed, err := s.feedRepo.GetByURL(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing feed: %w", err)
	}

	var feed *models.Feed
	isNewFeed := false

	if existingFeed != nil {
		// Feed already exists, check if user is already subscribed
		existingSubscription, err := s.subscriptionRepo.GetByUserAndFeed(ctx, userID, existingFeed.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing subscription: %w", err)
		}

		if existingSubscription != nil {
			return nil, fmt.Errorf("user is already subscribed to this feed")
		}

		feed = existingFeed
	} else {
		// New feed - validate and extract metadata
		metadata, err := s.ValidateAndExtractFeedMetadata(ctx, feedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to validate feed: %w", err)
		}

		// Create new feed
		feed = &models.Feed{
			ID:                   uuid.New(),
			FeedURL:              feedURL,
			Title:                &metadata.Title,
			Description:          &metadata.Description,
			SiteURL:              &metadata.SiteURL,
			PollingTier:          models.PollingTierActive, // Start with active tier
			NextPollAt:           time.Now(),               // Poll immediately
			Status:               models.FeedStatusActive,
			ConsecutiveErrorDays: 0,
		}

		if err := s.feedRepo.Create(ctx, feed); err != nil {
			return nil, fmt.Errorf("failed to create feed: %w", err)
		}

		isNewFeed = true
	}

	// Create subscription
	subscription := &models.FeedSubscription{
		ID:     uuid.New(),
		UserID: userID,
		FeedID: feed.ID,
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return &SubscriptionResult{
		Subscription: subscription,
		Feed:         feed,
		IsNewFeed:    isNewFeed,
	}, nil
}

// Unsubscribe unsubscribes a user from a feed
func (s *feedService) Unsubscribe(ctx context.Context, userID, feedID uuid.UUID) error {
	// Check if subscription exists
	subscription, err := s.subscriptionRepo.GetByUserAndFeed(ctx, userID, feedID)
	if err != nil {
		return fmt.Errorf("failed to check for subscription: %w", err)
	}

	if subscription == nil {
		return fmt.Errorf("subscription not found")
	}

	// Delete subscription
	if err := s.subscriptionRepo.DeleteByUserAndFeed(ctx, userID, feedID); err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

// ListUserSubscriptions lists all feeds a user is subscribed to
func (s *feedService) ListUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*UserFeedSubscription, error) {
	subscriptions, err := s.subscriptionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscriptions: %w", err)
	}

	result := make([]*UserFeedSubscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		feed, err := s.feedRepo.GetByID(ctx, subscription.FeedID)
		if err != nil {
			return nil, fmt.Errorf("failed to get feed: %w", err)
		}

		if feed == nil {
			// Skip if feed was deleted (shouldn't happen with CASCADE, but defensive)
			continue
		}

		result = append(result, &UserFeedSubscription{
			Subscription: subscription,
			Feed:         feed,
		})
	}

	return result, nil
}

// EnableFeed re-enables a disabled feed
func (s *feedService) EnableFeed(ctx context.Context, feedID uuid.UUID) error {
	feed, err := s.feedRepo.GetByID(ctx, feedID)
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	if feed == nil {
		return fmt.Errorf("feed not found")
	}

	if feed.Status == models.FeedStatusActive {
		return fmt.Errorf("feed is already active")
	}

	// Re-enable feed by setting status to active and scheduling next poll
	if err := s.feedRepo.UpdateStatus(ctx, feedID, models.FeedStatusActive); err != nil {
		return fmt.Errorf("failed to update feed status: %w", err)
	}

	// Reset error tracking and schedule next poll
	now := time.Now()
	if err := s.feedRepo.UpdateErrorInfo(ctx, feedID, 0, nil, nil); err != nil {
		return fmt.Errorf("failed to reset error info: %w", err)
	}

	// Update next poll time to now (will be polled soon)
	if err := s.feedRepo.UpdatePollingInfo(ctx, feedID, nil, nil, now, models.PollingTierActive); err != nil {
		return fmt.Errorf("failed to update polling info: %w", err)
	}

	return nil
}

// ValidateAndExtractFeedMetadata validates a feed URL and extracts metadata
func (s *feedService) ValidateAndExtractFeedMetadata(ctx context.Context, feedURL string) (*FeedMetadata, error) {
	// Create a context with timeout for the HTTP request
	reqCtx, cancel := context.WithTimeout(ctx, FeedFetchTimeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", fetch.UserAgent)

	// Fetch the feed
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	parsedFeed, err := parse.ParseReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Extract metadata
	metadata := &FeedMetadata{
		Title:       parsedFeed.Title,
		Description: parsedFeed.Description,
		SiteURL:     parsedFeed.SiteURL,
	}

	// Provide defaults if fields are empty
	if metadata.Title == "" {
		metadata.Title = "Untitled Feed"
	}
	if metadata.Description == "" {
		metadata.Description = ""
	}
	if metadata.SiteURL == "" {
		metadata.SiteURL = feedURL
	}

	return metadata, nil
}
