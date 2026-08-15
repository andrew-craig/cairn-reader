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

	"github.com/cairn-app/cairn-reader/pkg/logging"
	"github.com/google/uuid"
	"github.com/sony/gobreaker"
)

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

// bulkCreateItem is the per-item shape for POST /api/v1/content/bulk.
type bulkCreateItem struct {
	URL        string  `json:"url"`
	HTML       string  `json:"html"`
	SourceType string  `json:"source_type"`
	Title      *string `json:"title,omitempty"`
	Author     *string `json:"author,omitempty"`
}

// bulkCreateRequest is the body sent to POST /api/v1/content/bulk.
type bulkCreateRequest struct {
	Contents []bulkCreateItem `json:"contents"`
}

// bulkAddItem is the per-item shape for POST /api/v1/internal/content/user/bulk.
type bulkAddItem struct {
	ContentID uuid.UUID `json:"content_id"`
	UserID    uuid.UUID `json:"user_id"`
	Status    string    `json:"status,omitempty"`
}

// bulkAddRequest is the body sent to POST /api/v1/internal/content/user/bulk.
type bulkAddRequest struct {
	Items []bulkAddItem `json:"items"`
}

// createdItem holds the ID returned by the content creation endpoint.
type createdItem struct {
	ID uuid.UUID `json:"id"`
}

// failedItem represents a failed item in a bulk operation.
type failedItem struct {
	Index   int    `json:"index"`
	URL     string `json:"url"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// createResponseData is the payload inside the {"data": ...} envelope from POST /api/v1/content/bulk.
type createResponseData struct {
	Created  []createdItem `json:"created"`
	Existing []createdItem `json:"existing"`
	Failed   []failedItem  `json:"failed"`
}

// responseEnvelope unwraps the standard {"data": ..., "meta": ...} wrapper used by all services.
type responseEnvelope struct {
	Data json.RawMessage `json:"data"`
}

// ContentServiceConfig holds configuration for the client.
type ContentServiceConfig struct {
	BaseURL        string
	Timeout        time.Duration // default 30s
	InternalAPIKey string
}

// ContentServiceClient delivers processed email content to the Content Service.
type ContentServiceClient struct {
	baseURL        string
	internalAPIKey string
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
		baseURL:        strings.TrimRight(cfg.BaseURL, "/"),
		internalAPIKey: cfg.InternalAPIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
	}
}

// DeliverContent creates the content on the Content Service then links it to the user.
// Returns the content service ID on success. On failure, the caller is responsible for
// retrying — the outbox's DB-level backoff schedule (see outbox_worker.go) owns retry
// timing so that one failing entry never blocks the rest of a batch.
func (c *ContentServiceClient) DeliverContent(ctx context.Context, payload []EmailContentItem) (uuid.UUID, error) {
	if c.internalAPIKey == "" {
		return uuid.Nil, fmt.Errorf("internal API key is required but not configured")
	}
	if len(payload) == 0 {
		return uuid.Nil, fmt.Errorf("payload must not be empty")
	}

	id, err := c.attemptDeliver(ctx, payload)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (c *ContentServiceClient) attemptDeliver(ctx context.Context, payload []EmailContentItem) (uuid.UUID, error) {
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.doDeliver(ctx, payload)
	})
	if err != nil {
		return uuid.Nil, err
	}
	id, _ := result.(uuid.UUID)
	return id, nil
}

// doDeliver performs the two-step delivery: create content, then link to user.
func (c *ContentServiceClient) doDeliver(ctx context.Context, payload []EmailContentItem) (uuid.UUID, error) {
	contentID, err := c.createContent(ctx, payload)
	if err != nil {
		return uuid.Nil, err
	}
	if err := c.addContentToUsers(ctx, contentID, payload); err != nil {
		return uuid.Nil, err
	}
	return contentID, nil
}

// createContent calls POST /api/v1/content/bulk to create or deduplicate the email content.
// Returns the content ID from the first created or existing item.
func (c *ContentServiceClient) createContent(ctx context.Context, payload []EmailContentItem) (uuid.UUID, error) {
	items := make([]bulkCreateItem, len(payload))
	for i := range payload {
		p := &payload[i]
		item := bulkCreateItem{URL: p.URL, HTML: p.HTML, SourceType: p.SourceType}
		if p.Title != "" {
			item.Title = &p.Title
		}
		if p.Author != "" {
			item.Author = &p.Author
		}
		items[i] = item
	}

	body, err := json.Marshal(bulkCreateRequest{Contents: items})
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal content request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/content/bulk", bytes.NewBuffer(body))
	if err != nil {
		return uuid.Nil, fmt.Errorf("build content request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-API-Key", c.internalAPIKey)
	logging.SetRequestIDHeader(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("content create request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return uuid.Nil, fmt.Errorf("read create response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return uuid.Nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var env responseEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return uuid.Nil, fmt.Errorf("decode create envelope: %w", err)
	}
	var data createResponseData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return uuid.Nil, fmt.Errorf("decode create response: %w", err)
	}

	if len(data.Created) > 0 {
		return data.Created[0].ID, nil
	}
	if len(data.Existing) > 0 {
		return data.Existing[0].ID, nil
	}
	return uuid.Nil, fmt.Errorf("no content ID returned from create")
}

// addContentToUsers calls POST /api/v1/internal/content/user/bulk to link content to users.
func (c *ContentServiceClient) addContentToUsers(ctx context.Context, contentID uuid.UUID, payload []EmailContentItem) error {
	items := make([]bulkAddItem, len(payload))
	for i, p := range payload {
		items[i] = bulkAddItem{ContentID: contentID, UserID: p.UserID, Status: "unread"}
	}

	body, err := json.Marshal(bulkAddRequest{Items: items})
	if err != nil {
		return fmt.Errorf("marshal add-to-users request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/internal/content/user/bulk", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("build add-to-users request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Internal-API-Key", c.internalAPIKey)
	logging.SetRequestIDHeader(ctx, req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("add-to-users request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read add-to-users response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
