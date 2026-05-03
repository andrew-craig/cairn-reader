package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/processor"
)

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

// Validate validates the BulkContentItem
func (b BulkContentItem) Validate() error {
	return validation.ValidateStruct(&b,
		validation.Field(&b.URL,
			validation.Required.Error("URL is required"),
			is.URL.Error("URL must be a valid URL"),
		),
		validation.Field(&b.HTML,
			validation.Required.Error("HTML is required"),
			validation.Length(0, processor.MaxContentSize).
				Error("HTML exceeds the 5MB per-item maximum"),
		),
		validation.Field(&b.SourceType,
			validation.Required.Error("source_type is required"),
			validation.In(models.SourceTypeRSS, models.SourceTypeWeb, models.SourceTypeEmail).
				Error("Invalid source_type. Must be 'rss', 'web', or 'email'"),
		),
		validation.Field(&b.Title,
			validation.When(b.Title != nil,
				validation.Length(0, 1000).Error("title must be 1000 characters or fewer"),
			),
		),
		validation.Field(&b.Author,
			validation.When(b.Author != nil,
				validation.Length(0, 255).Error("author must be 255 characters or fewer"),
			),
		),
	)
}

// BulkCreateContentRequest represents the request body for batch creating contents
type BulkCreateContentRequest struct {
	Contents []BulkContentItem `json:"contents"`
}

// Validate validates the BulkCreateContentRequest
func (b BulkCreateContentRequest) Validate() error {
	return validation.ValidateStruct(&b,
		validation.Field(&b.Contents,
			validation.Required.Error("contents is required"),
			validation.Length(1, 100).Error("contents array must have 1-100 items"),
			validation.Each(validation.By(func(value interface{}) error {
				item, ok := value.(BulkContentItem)
				if !ok {
					return validation.NewError("validation_type_error", "invalid item type")
				}
				return item.Validate()
			})),
		),
	)
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

// Validate validates the CheckDuplicatesRequest
func (c CheckDuplicatesRequest) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.Items,
			validation.Required.Error("items is required"),
			validation.Length(1, 100).Error("items array must have 1-100 items"),
		),
	)
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

// Validate validates the BulkAddToUsersItem
func (b BulkAddToUsersItem) Validate() error {
	return validation.ValidateStruct(&b,
		validation.Field(&b.Status,
			validation.When(b.Status != "",
				validation.In(models.StatusUnread, models.StatusRead, models.StatusArchived).
					Error("Invalid status. Must be 'unread', 'read', or 'archived'"),
			),
		),
	)
}

// BulkAddToUsersRequest represents the request body for batch adding contents to users
type BulkAddToUsersRequest struct {
	Items []BulkAddToUsersItem `json:"items"`
}

// Validate validates the BulkAddToUsersRequest
func (b BulkAddToUsersRequest) Validate() error {
	return validation.ValidateStruct(&b,
		validation.Field(&b.Items,
			validation.Required.Error("items is required"),
			validation.Length(1, 100).Error("items array must have 1-100 items"),
			validation.Each(validation.By(func(value interface{}) error {
				item, ok := value.(BulkAddToUsersItem)
				if !ok {
					return validation.NewError("validation_type_error", "invalid item type")
				}
				return item.Validate()
			})),
		),
	)
}

// BulkAddToUsersResponse represents the response for batch adding contents to users
type BulkAddToUsersResponse struct {
	Succeeded []uuid.UUID      `json:"succeeded"` // List of successfully created user_content IDs
	Failed    []BulkFailedItem `json:"failed"`
}
