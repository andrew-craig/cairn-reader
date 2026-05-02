package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sony/gobreaker"
)

// ContentServiceClient is an HTTP client for the Content Service API
type ContentServiceClient struct {
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
	maxRetries     int
	retryDelay     time.Duration
	internalAPIKey string
}

// ContentServiceConfig holds configuration for the Content Service client
type ContentServiceConfig struct {
	BaseURL        string
	Timeout        time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
	InternalAPIKey string
}

// CreateContentRequest represents the request body for creating content
type CreateContentRequest struct {
	URL          string     `json:"url"`
	HTML         *string    `json:"html,omitempty"`
	SourceType   string     `json:"source_type"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

// UpdateContentRequest represents the request body for updating content
type UpdateContentRequest struct {
	URL         string     `json:"url"`
	HTML        string     `json:"html"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// BulkContentItem represents a single content item in a bulk request
type BulkContentItem struct {
	URL          string     `json:"url"`
	HTML         string     `json:"html"`
	SourceType   string     `json:"source_type"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	Title        *string    `json:"title,omitempty"`
	Author       *string    `json:"author,omitempty"`
}

// BulkCreateContentRequest represents the request body for batch creating contents
type BulkCreateContentRequest struct {
	Contents []BulkContentItem `json:"contents"`
}

// ContentResponse represents a content item in API responses
type ContentResponse struct {
	ID           uuid.UUID  `json:"id"`
	ContentHash  string     `json:"content_hash"`
	CleanedHTML  string     `json:"cleaned_html"`
	OriginalURL  string     `json:"original_url"`
	CanonicalURL *string    `json:"canonical_url,omitempty"`
	Title        string     `json:"title"`
	Author       *string    `json:"author,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	Description  *string    `json:"description,omitempty"`
	ImageURLs    []string   `json:"image_urls,omitempty"`
	SourceType   string     `json:"source_type"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// BulkCreateContentResponse represents the response for batch creating contents
type BulkCreateContentResponse struct {
	Created  []*ContentResponse `json:"created"`
	Existing []*ContentResponse `json:"existing"`
	Failed   []BulkFailedItem   `json:"failed"`
}

// BulkFailedItem represents a failed item in a bulk operation
type BulkFailedItem struct {
	Index   int    `json:"index"`
	URL     string `json:"url"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CheckDuplicatesItem represents a single item to check for duplicates
type CheckDuplicatesItem struct {
	ContentHash  string    `json:"content_hash"`
	SourceFeedID uuid.UUID `json:"source_feed_id"`
}

// CheckDuplicatesRequest represents the request body for checking duplicates
type CheckDuplicatesRequest struct {
	Items []CheckDuplicatesItem `json:"items"`
}

// DuplicateCheckResult represents the result of a duplicate check
type DuplicateCheckResult struct {
	ContentHash string           `json:"content_hash"`
	Exists      bool             `json:"exists"`
	ContentID   *uuid.UUID       `json:"content_id,omitempty"`
	Content     *ContentResponse `json:"content,omitempty"`
}

// CheckDuplicatesResponse represents the response for checking duplicates
type CheckDuplicatesResponse struct {
	Results []DuplicateCheckResult `json:"results"`
}

// BulkAddToUsersItem represents a content to add to a user
type BulkAddToUsersItem struct {
	ContentID uuid.UUID `json:"content_id"`
	UserID    uuid.UUID `json:"user_id"`
	Status    string    `json:"status,omitempty"` // defaults to "unread"
}

// BulkAddToUsersRequest represents the request body for batch adding contents to users
type BulkAddToUsersRequest struct {
	Items []BulkAddToUsersItem `json:"items"`
}

// BulkAddToUsersResponse represents the response for batch adding contents to users
type BulkAddToUsersResponse struct {
	Succeeded []uuid.UUID      `json:"succeeded"` // List of successfully created user_content IDs
	Failed    []BulkFailedItem `json:"failed"`
}

// ErrorResponse represents an error response from the Content Service
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// NewContentServiceClient creates a new Content Service client
func NewContentServiceClient(config ContentServiceConfig) *ContentServiceClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	// Configure circuit breaker
	// Opens after 5 consecutive failures
	// Half-opens after 30 seconds
	cbSettings := gobreaker.Settings{
		Name:        "ContentService",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log state changes (could be integrated with structured logging)
			fmt.Printf("Circuit breaker state changed from %s to %s\n", from, to)
		},
	}

	return &ContentServiceClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		circuitBreaker: gobreaker.NewCircuitBreaker(cbSettings),
		maxRetries:     config.MaxRetries,
		retryDelay:     config.RetryDelay,
		internalAPIKey: config.InternalAPIKey,
	}
}

// CreateContent creates a single content item
func (c *ContentServiceClient) CreateContent(ctx context.Context, req CreateContentRequest) (*ContentResponse, error) {
	var result ContentResponse
	err := c.doWithRetry(ctx, "POST", "/api/v1/content", req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create content: %w", err)
	}
	return &result, nil
}

// UpdateContent updates an existing content item (preserves user relationships)
func (c *ContentServiceClient) UpdateContent(ctx context.Context, contentID uuid.UUID, req UpdateContentRequest) (*ContentResponse, error) {
	var result ContentResponse
	path := fmt.Sprintf("/api/v1/content/%s", contentID)
	err := c.doWithRetry(ctx, "PUT", path, req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to update content: %w", err)
	}
	return &result, nil
}

// BulkCreateContent batch creates up to 100 content items
func (c *ContentServiceClient) BulkCreateContent(ctx context.Context, contents []BulkContentItem) (*BulkCreateContentResponse, error) {
	if len(contents) == 0 {
		return nil, fmt.Errorf("contents array cannot be empty")
	}
	if len(contents) > 100 {
		return nil, fmt.Errorf("maximum 100 items allowed per bulk request")
	}

	req := BulkCreateContentRequest{
		Contents: contents,
	}

	var result BulkCreateContentResponse
	err := c.doWithRetry(ctx, "POST", "/api/v1/content/bulk", req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk create content: %w", err)
	}
	return &result, nil
}

// CheckDuplicates checks for existing content by feed ID and content hashes
func (c *ContentServiceClient) CheckDuplicates(ctx context.Context, items []CheckDuplicatesItem) (*CheckDuplicatesResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("items array cannot be empty")
	}
	if len(items) > 100 {
		return nil, fmt.Errorf("maximum 100 items allowed per request")
	}

	req := CheckDuplicatesRequest{
		Items: items,
	}

	var result CheckDuplicatesResponse
	err := c.doWithRetry(ctx, "POST", "/api/v1/content/check-duplicate", req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicates: %w", err)
	}
	return &result, nil
}

// AddContentToUsers adds content to multiple users' reading lists
func (c *ContentServiceClient) AddContentToUsers(ctx context.Context, items []BulkAddToUsersItem) (*BulkAddToUsersResponse, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("items array cannot be empty")
	}
	if len(items) > 100 {
		return nil, fmt.Errorf("maximum 100 items allowed per request")
	}

	req := BulkAddToUsersRequest{
		Items: items,
	}

	var result BulkAddToUsersResponse
	err := c.doWithRetry(ctx, "POST", "/api/v1/internal/content/user/bulk", req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to add content to users: %w", err)
	}
	return &result, nil
}

// doWithRetry performs an HTTP request with retry logic and circuit breaker
func (c *ContentServiceClient) doWithRetry(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: delay * 2^attempt
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Execute request through circuit breaker
		_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
			return nil, c.doRequest(ctx, method, path, body, result)
		})

		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on certain errors
		if isNonRetryableError(err) {
			return err
		}

		// Don't retry if context is done
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs the actual HTTP request
func (c *ContentServiceClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.internalAPIKey != "" {
		req.Header.Set("X-Internal-API-Key", c.internalAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle HTTP errors
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("HTTP %d: %s - %s", resp.StatusCode, errResp.Error, errResp.Message)
	}

	// Decode response. Content Service wraps successful responses in a
	// {"data": ..., "meta": ...} envelope (see pkg/api/response.go), so we
	// unwrap the envelope before unmarshalling into the caller's result.
	if result != nil {
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(respBody, &envelope); err != nil {
			return fmt.Errorf("failed to decode response envelope: %w", err)
		}
		if len(envelope.Data) == 0 {
			return fmt.Errorf("response missing 'data' field: %.100s", string(respBody))
		}
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}

	return nil
}

// isNonRetryableError returns true if the error should not be retried
func isNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Don't retry 4xx errors (except 429 Too Many Requests)
	// These typically indicate client errors that won't be fixed by retrying
	if contains(errMsg, "HTTP 400") ||
		contains(errMsg, "HTTP 401") ||
		contains(errMsg, "HTTP 403") ||
		contains(errMsg, "HTTP 404") ||
		contains(errMsg, "HTTP 422") {
		return true
	}

	// Don't retry validation errors
	if contains(errMsg, "validation_error") {
		return true
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(findSubstring(s, substr) != -1))
}

// findSubstring finds the index of substr in s
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
