package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/logging"
)

// ErrSubscriptionNotFound is returned when an unsubscribe targets a subscription
// that does not exist (already unsubscribed or never subscribed).
var ErrSubscriptionNotFound = errors.New("subscription not found")

// IngestRSSClient handles communication with the Ingest RSS service
type IngestRSSClient struct {
	baseURL        string
	internalAPIKey string
	httpClient     *http.Client
}

// NewIngestRSSClient creates a new Ingest RSS service client
func NewIngestRSSClient(baseURL, internalAPIKey string) *IngestRSSClient {
	return &IngestRSSClient{
		baseURL:        baseURL,
		internalAPIKey: internalAPIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SubscribeFeedRequest represents the request to subscribe to a feed
type SubscribeFeedRequest struct {
	FeedURL string `json:"feed_url"`
}

// FeedSubscriptionResponse represents a feed subscription from Ingest RSS.
// Fields match the flat response from the Ingest RSS service's subscribe endpoint.
type FeedSubscriptionResponse struct {
	SubscriptionID string    `json:"subscription_id"`
	FeedID         string    `json:"feed_id"`
	FeedURL        string    `json:"feed_url"`
	FeedTitle      string    `json:"feed_title"`
	IsNewFeed      bool      `json:"is_new_feed"`
	SubscribedAt   time.Time `json:"subscribed_at"`
}

// IngestRSSError represents an error from the Ingest RSS service
type IngestRSSError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SubscriptionInfo represents subscription information in list.
// Fields match the flat response from the Ingest RSS service's list endpoint.
type SubscriptionInfo struct {
	ID            string `json:"subscription_id"`
	FeedID        string `json:"feed_id"`
	FeedURL       string `json:"feed_url"`
	FeedTitle     string `json:"feed_title"`
	FeedStatus    string `json:"feed_status"`
	PollingTier   string `json:"polling_tier"`
	LastFetchedAt string `json:"last_fetched_at"`
	SubscribedAt  string `json:"subscribed_at"`
}

// ListSubscriptionsResponse represents the response from listing subscriptions
type ListSubscriptionsResponse struct {
	Subscriptions []SubscriptionInfo `json:"subscriptions"`
	Count         int                `json:"count"`
}

// SubscribeUserToFeed subscribes a user to an RSS feed
func (c *IngestRSSClient) SubscribeUserToFeed(ctx context.Context, userID, feedURL string) (*FeedSubscriptionResponse, error) {
	reqBody := SubscribeFeedRequest{
		FeedURL: feedURL,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/source/rss/user/%s/subscription", c.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", c.internalAPIKey)
	logging.SetRequestIDHeader(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ingest RSS service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiError IngestRSSError
		if err := json.Unmarshal(body, &apiError); err != nil {
			return nil, fmt.Errorf("ingest RSS service error (status %d): %s", resp.StatusCode, string(body))
		}

		// Map specific error codes
		switch resp.StatusCode {
		case http.StatusConflict:
			if apiError.Error == "already_subscribed" {
				return nil, fmt.Errorf("already subscribed to this feed")
			}
		case http.StatusBadRequest:
			if apiError.Error == "feed_limit_reached" {
				return nil, fmt.Errorf("feed limit reached (max 100 feeds per user)")
			}
			if apiError.Error == "invalid_feed" {
				return nil, fmt.Errorf("invalid feed URL or not a valid RSS/Atom feed")
			}
		}

		return nil, fmt.Errorf("ingest RSS service error: %s", apiError.Message)
	}

	// Parse the API response wrapper
	var apiResponse struct {
		Data FeedSubscriptionResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResponse.Data, nil
}

// UnsubscribeUserFromFeed removes a user's subscription to an RSS feed.
// It returns nil on success, ErrSubscriptionNotFound if no such subscription exists.
func (c *IngestRSSClient) UnsubscribeUserFromFeed(ctx context.Context, userID, feedID string) error {
	url := fmt.Sprintf("%s/api/v1/source/rss/user/%s/subscription/%s", c.baseURL, userID, feedID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Internal-API-Key", c.internalAPIKey)
	logging.SetRequestIDHeader(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call ingest RSS service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return ErrSubscriptionNotFound
	}

	const maxErrorBodySize = 4096
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiError IngestRSSError
	if err := json.Unmarshal(body, &apiError); err != nil {
		return fmt.Errorf("ingest RSS service error (status %d): %s", resp.StatusCode, string(body))
	}
	return fmt.Errorf("ingest RSS service error: %s", apiError.Message)
}

// ListUserSubscriptions fetches RSS subscriptions for a user from Ingest RSS service
func (c *IngestRSSClient) ListUserSubscriptions(ctx context.Context, userID string) (*ListSubscriptionsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/source/rss/user/%s/subscription", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Internal-API-Key", c.internalAPIKey)
	logging.SetRequestIDHeader(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscriptions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiError IngestRSSError
		if err := json.Unmarshal(body, &apiError); err != nil {
			return nil, fmt.Errorf("ingest RSS service error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("ingest RSS service error: %s", apiError.Message)
	}

	// Parse the API response wrapper
	var apiResponse struct {
		Data ListSubscriptionsResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResponse.Data, nil
}
