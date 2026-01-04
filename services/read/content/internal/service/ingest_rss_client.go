package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// IngestRSSClient handles communication with the Ingest RSS service
type IngestRSSClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewIngestRSSClient creates a new Ingest RSS service client
func NewIngestRSSClient(baseURL string) *IngestRSSClient {
	return &IngestRSSClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SubscribeFeedRequest represents the request to subscribe to a feed
type SubscribeFeedRequest struct {
	FeedURL string `json:"feed_url"`
}

// FeedSubscriptionResponse represents a feed subscription from Ingest RSS
type FeedSubscriptionResponse struct {
	Subscription struct {
		ID           string    `json:"id"`
		UserID       string    `json:"user_id"`
		FeedID       string    `json:"feed_id"`
		SubscribedAt time.Time `json:"subscribed_at"`
	} `json:"subscription"`
	Feed struct {
		ID          string    `json:"id"`
		FeedURL     string    `json:"feed_url"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		SiteURL     string    `json:"site_url"`
		PollingTier string    `json:"polling_tier"`
		Status      string    `json:"status"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"feed"`
	IsNewFeed bool `json:"is_new_feed"`
}

// IngestRSSError represents an error from the Ingest RSS service
type IngestRSSError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ingest RSS service: %w", err)
	}
	defer resp.Body.Close()

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

	// Parse success response
	var subscription FeedSubscriptionResponse
	if err := json.Unmarshal(body, &subscription); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &subscription, nil
}
