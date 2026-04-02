// Package client provides HTTP clients for external service communication.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sony/gobreaker"
)

// retryDelays defines the fixed backoff schedule for delivery retries.
// Mirrors the outbox retry schedule: 1m, 5m, 15m, 1h, 4h, 12h.
var retryDelays = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	1 * time.Hour,
	4 * time.Hour,
	12 * time.Hour,
}

// EmailContentItem represents a single email content item in a bulk delivery request.
type EmailContentItem struct {
	UserID     uuid.UUID `json:"user_id"`
	URL        string    `json:"url"`
	Type       string    `json:"type"`
	HTML       string    `json:"html"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	SourceType string    `json:"source_type"`
}

// deliverResponse is the shape returned by the internal bulk endpoint.
type deliverResponse struct {
	Created  []createdItem `json:"created"`
	Existing []createdItem `json:"existing"`
	Failed   []failedItem  `json:"failed"`
}

type createdItem struct {
	ID uuid.UUID `json:"id"`
}

type failedItem struct {
	Index   int    `json:"index"`
	URL     string `json:"url"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ContentServiceConfig holds configuration for the client.
type ContentServiceConfig struct {
	BaseURL string
	Timeout time.Duration // default 30s
}

// ContentServiceClient delivers processed email content to the Content Service.
type ContentServiceClient struct {
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewContentServiceClient creates a new client. Timeout defaults to 30s.
func NewContentServiceClient(cfg ContentServiceConfig) *ContentServiceClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cbSettings := gobreaker.Settings{
		Name:        "EmailContentService",
		MaxRequests: 1,
		Timeout:     30 * time.Second, // half-open after 30s
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			fmt.Printf("circuit breaker %q: %s → %s\n", name, from, to)
		},
	}

	return &ContentServiceClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
	}
}

// DeliverContent posts the email content items to the Content Service internal
// bulk endpoint and returns the content service ID of the first created item.
func (c *ContentServiceClient) DeliverContent(ctx context.Context, payload []EmailContentItem) (uuid.UUID, error) {
	if len(payload) == 0 {
		return uuid.Nil, fmt.Errorf("payload must not be empty")
	}

	var lastErr error
	// Initial attempt + one entry per retry delay.
	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		if attempt > 0 {
			delay := retryDelays[attempt-1]
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return uuid.Nil, ctx.Err()
			}
		}

		id, err := c.attemptDeliver(ctx, payload)
		if err == nil {
			return id, nil
		}
		lastErr = err

		if isNonRetryable(err) || ctx.Err() != nil {
			break
		}
	}

	return uuid.Nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// attemptDeliver performs a single delivery attempt through the circuit breaker.
func (c *ContentServiceClient) attemptDeliver(ctx context.Context, payload []EmailContentItem) (uuid.UUID, error) {
	result, cbErr := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.doRequest(ctx, payload)
	})
	if cbErr != nil {
		return uuid.Nil, cbErr
	}
	resp, ok := result.(*deliverResponse)
	if !ok || len(resp.Created) == 0 {
		// All items may already exist.
		if len(resp.Existing) > 0 {
			return resp.Existing[0].ID, nil
		}
		return uuid.Nil, fmt.Errorf("no content IDs returned")
	}
	return resp.Created[0].ID, nil
}

// doRequest marshals the payload and executes the HTTP POST.
func (c *ContentServiceClient) doRequest(ctx context.Context, payload []EmailContentItem) (*deliverResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := c.baseURL + "/api/v1/internal/content/user/bulk"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result deliverResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// isNonRetryable returns true for 4xx errors that won't be resolved by retrying.
func isNonRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, code := range []string{"HTTP 400", "HTTP 401", "HTTP 403", "HTTP 404", "HTTP 422"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}
