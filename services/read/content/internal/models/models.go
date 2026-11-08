package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Content represents a unique content item (shared across users)
type Content struct {
	ID           uuid.UUID      `json:"id"`
	ContentHash  string         `json:"content_hash"`
	CleanedHTML  string         `json:"cleaned_html"`
	OriginalURL  string         `json:"original_url"`
	CanonicalURL *string        `json:"canonical_url,omitempty"`
	Title        string         `json:"title"`
	Author       *string        `json:"author,omitempty"`
	PublishedAt  *time.Time     `json:"published_at,omitempty"`
	Description  *string        `json:"description,omitempty"`
	ImageURLs    pq.StringArray `json:"image_urls,omitempty"`
	SourceType   string         `json:"source_type"`
	SourceFeedID *uuid.UUID     `json:"source_feed_id,omitempty"`
	// SourceSenderID is the email sender that produced the content (email only).
	SourceSenderID *uuid.UUID `json:"source_sender_id,omitempty"`
	Metadata       JSONB      `json:"metadata,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	OrphanedAt     *time.Time `json:"orphaned_at,omitempty"`

	// Source is the raw extraction input. Repository writes persist it to
	// content_sources in the same transaction as the content write when set;
	// reads never populate it (use ContentRepository.GetSource).
	Source *ContentSource `json:"-"`
}

// ContentSource is the raw input of a content's extraction (content_sources row).
type ContentSource struct {
	ContentID         uuid.UUID
	RawHTML           string
	ReadabilityFailed bool
	CreatedAt         time.Time
}

// Extraction feedback statuses.
const (
	FeedbackStatusOpen     = "open"
	FeedbackStatusResolved = "resolved"
)

// FeedbackCategories are the extraction outputs a user can flag. Mirrors the
// chk_feedback_categories constraint in migration 000007.
var FeedbackCategories = []string{
	"body_missing_content", "body_extra_content", "images", "formatting", "links",
	"title", "author", "description", "lead_image", "other",
}

// ContentFeedback is a user's report of a bad extraction (content_feedback row).
// Snapshot is only set when writing; reads never populate it.
type ContentFeedback struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	ContentID  uuid.UUID
	Categories pq.StringArray
	Comment    *string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Snapshot   *FeedbackSnapshot
}

// FeedbackSnapshot is the evidence copied from contents and content_sources
// when a report is submitted. RawHTML and ReadabilityFailed are nil when the
// content_sources row had expired.
type FeedbackSnapshot struct {
	OriginalURL       string
	Title             string
	Author            *string
	Description       *string
	ImageURL          *string
	CleanedHTML       string
	RawHTML           *string
	ReadabilityFailed *bool
}

// UserContent represents the junction table mapping users to content with user-specific metadata
type UserContent struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	ContentID      uuid.UUID `json:"content_id"`
	Status         string    `json:"status"`
	List           string    `json:"list"`
	ScrollPosition float64   `json:"scroll_position"`
	IsFavorite     bool      `json:"is_favorite"`
	AddedAt        time.Time `json:"added_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// JSONB is a custom type for handling PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	result := make(JSONB)
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

// ContentStatus constants
const (
	StatusUnread    = "unread"
	StatusReading   = "reading"
	StatusCompleted = "completed"
	StatusArchived  = "archived"
)

// SourceType constants
const (
	SourceTypeRSS   = "rss"
	SourceTypeWeb   = "web"
	SourceTypeEmail = "email"
)

// List constants: the two destinations a user_contents row can live in.
const (
	ListFeed  = "feed"
	ListReads = "reads"
)

// SourceRoute is a user's chosen destination list for one source
// (SourceKey is the feed ID for rss, the sender ID for email).
type SourceRoute struct {
	UserID     uuid.UUID `json:"user_id"`
	SourceType string    `json:"source_type"`
	SourceKey  uuid.UUID `json:"source_key"`
	List       string    `json:"list"`
}

// ValidateList checks if the given list is valid
func ValidateList(list string) bool {
	return list == ListFeed || list == ListReads
}

// ValidateStatus checks if the given status is valid
func ValidateStatus(status string) bool {
	return status == StatusUnread || status == StatusReading || status == StatusCompleted || status == StatusArchived
}
