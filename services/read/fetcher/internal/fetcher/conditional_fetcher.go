package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
)

// ConditionalFetchResult represents the result of a conditional fetch
type ConditionalFetchResult struct {
	// NotModified indicates if the content has not been modified (HTTP 304)
	NotModified bool
	// Content is the fetched content (nil if NotModified)
	Content []byte
	// LastModified is the Last-Modified header from the response
	LastModified *string
	// ETag is the ETag header from the response
	ETag *string
	// StatusCode is the HTTP status code
	StatusCode int
}

// ConditionalFetcher handles HTTP requests with conditional headers
type ConditionalFetcher struct {
	httpClient *http.Client
}

// NewConditionalFetcher creates a new conditional fetcher
func NewConditionalFetcher(httpClient *http.Client) *ConditionalFetcher {
	return &ConditionalFetcher{
		httpClient: httpClient,
	}
}

// FetchWithConditionals performs a conditional HTTP request
func (cf *ConditionalFetcher) FetchWithConditionals(
	ctx context.Context,
	url string,
	lastModified *string,
	etag *string,
) (*ConditionalFetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Cairn-RSS-Fetcher/1.0")

	// Add conditional headers if available
	if lastModified != nil && *lastModified != "" {
		req.Header.Set("If-Modified-Since", *lastModified)
	}
	if etag != nil && *etag != "" {
		req.Header.Set("If-None-Match", *etag)
	}

	resp, err := cf.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	result := &ConditionalFetchResult{
		StatusCode:  resp.StatusCode,
		NotModified: resp.StatusCode == http.StatusNotModified,
	}

	// Extract caching headers from response
	if lastModifiedHeader := resp.Header.Get("Last-Modified"); lastModifiedHeader != "" {
		result.LastModified = &lastModifiedHeader
	}
	if etagHeader := resp.Header.Get("ETag"); etagHeader != "" {
		result.ETag = &etagHeader
	}

	// If not modified, return early
	if result.NotModified {
		return result, nil
	}

	// Check for successful status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result.Content = content
	return result, nil
}

// FetchForItem performs a conditional fetch for a feed item
func (cf *ConditionalFetcher) FetchForItem(
	ctx context.Context,
	item *models.FeedItem,
) (*ConditionalFetchResult, error) {
	return cf.FetchWithConditionals(
		ctx,
		item.ItemURL,
		item.HTTPLastModified,
		item.HTTPETag,
	)
}
