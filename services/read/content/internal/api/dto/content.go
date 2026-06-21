package dto

import (
	"time"

	"github.com/google/uuid"
)

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

// ContentResponse represents a content item in API responses (full detail, includes cleaned_html)
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
	WordCount    int        `json:"word_count,omitempty"`
	SourceType   string     `json:"source_type"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ContentSummaryResponse is a lightweight content item used in list/search responses.
// It omits cleaned_html to avoid sending large payloads for items the user may not open.
type ContentSummaryResponse struct {
	ID           uuid.UUID  `json:"id"`
	ContentHash  string     `json:"content_hash"`
	OriginalURL  string     `json:"original_url"`
	CanonicalURL *string    `json:"canonical_url,omitempty"`
	Title        string     `json:"title"`
	Author       *string    `json:"author,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	Description  *string    `json:"description,omitempty"`
	ImageURLs    []string   `json:"image_urls,omitempty"`
	WordCount    int        `json:"word_count,omitempty"`
	SourceType   string     `json:"source_type"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}
