package dto

import (
	"time"

	"github.com/google/uuid"
)

// BulkContentItem represents a single content item in a bulk request
type BulkContentItem struct {
	URL          string     `json:"url"`
	HTML         string     `json:"html"`
	SourceType   string     `json:"source_type"`
	SourceFeedID *uuid.UUID `json:"source_feed_id,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
}

// BulkCreateContentRequest represents the request body for batch creating contents
type BulkCreateContentRequest struct {
	Contents []BulkContentItem `json:"contents"`
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
	ContentHash  string         `json:"content_hash"`
	Exists       bool           `json:"exists"`
	ContentID    *uuid.UUID     `json:"content_id,omitempty"`
	Content      *ContentResponse `json:"content,omitempty"`
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
