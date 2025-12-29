package dto

import (
	"time"

	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-read/fetcher/internal/service"
	"github.com/google/uuid"
)

// SubscribeFeedRequest represents a request to subscribe to a feed
type SubscribeFeedRequest struct {
	FeedURL string `json:"feed_url"`
}

// SubscribeFeedResponse represents the response after subscribing to a feed
type SubscribeFeedResponse struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	FeedID         uuid.UUID `json:"feed_id"`
	FeedURL        string    `json:"feed_url"`
	FeedTitle      string    `json:"feed_title"`
	IsNewFeed      bool      `json:"is_new_feed"`
	SubscribedAt   time.Time `json:"subscribed_at"`
}

// FeedSubscriptionResponse represents a single feed subscription
type FeedSubscriptionResponse struct {
	SubscriptionID uuid.UUID              `json:"subscription_id"`
	FeedID         uuid.UUID              `json:"feed_id"`
	FeedURL        string                 `json:"feed_url"`
	FeedTitle      string                 `json:"feed_title"`
	FeedStatus     models.FeedStatus      `json:"feed_status"`
	PollingTier    models.PollingTier     `json:"polling_tier"`
	LastFetchedAt  *time.Time             `json:"last_fetched_at,omitempty"`
	SubscribedAt   time.Time              `json:"subscribed_at"`
}

// ListFeedSubscriptionsResponse represents the response for listing subscriptions
type ListFeedSubscriptionsResponse struct {
	Subscriptions []FeedSubscriptionResponse `json:"subscriptions"`
	Count         int                        `json:"count"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// ToSubscribeFeedResponse converts a SubscriptionResult to a SubscribeFeedResponse
func ToSubscribeFeedResponse(result *service.SubscriptionResult) SubscribeFeedResponse {
	feedTitle := "Untitled Feed"
	if result.Feed.Title != nil {
		feedTitle = *result.Feed.Title
	}

	return SubscribeFeedResponse{
		SubscriptionID: result.Subscription.ID,
		FeedID:         result.Feed.ID,
		FeedURL:        result.Feed.FeedURL,
		FeedTitle:      feedTitle,
		IsNewFeed:      result.IsNewFeed,
		SubscribedAt:   result.Subscription.SubscribedAt,
	}
}

// ToFeedSubscriptionResponse converts a UserFeedSubscription to a FeedSubscriptionResponse
func ToFeedSubscriptionResponse(userSub *service.UserFeedSubscription) FeedSubscriptionResponse {
	feedTitle := "Untitled Feed"
	if userSub.Feed.Title != nil {
		feedTitle = *userSub.Feed.Title
	}

	return FeedSubscriptionResponse{
		SubscriptionID: userSub.Subscription.ID,
		FeedID:         userSub.Feed.ID,
		FeedURL:        userSub.Feed.FeedURL,
		FeedTitle:      feedTitle,
		FeedStatus:     userSub.Feed.Status,
		PollingTier:    userSub.Feed.PollingTier,
		LastFetchedAt:  userSub.Feed.LastFetchedAt,
		SubscribedAt:   userSub.Subscription.SubscribedAt,
	}
}

// ToListFeedSubscriptionsResponse converts a list of UserFeedSubscriptions to a ListFeedSubscriptionsResponse
func ToListFeedSubscriptionsResponse(userSubs []*service.UserFeedSubscription) ListFeedSubscriptionsResponse {
	subscriptions := make([]FeedSubscriptionResponse, 0, len(userSubs))
	for _, userSub := range userSubs {
		subscriptions = append(subscriptions, ToFeedSubscriptionResponse(userSub))
	}

	return ListFeedSubscriptionsResponse{
		Subscriptions: subscriptions,
		Count:         len(subscriptions),
	}
}
