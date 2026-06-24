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
	Metadata     JSONB          `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	OrphanedAt   *time.Time     `json:"orphaned_at,omitempty"`
}

// UserContent represents the junction table mapping users to content with user-specific metadata
type UserContent struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	ContentID      uuid.UUID `json:"content_id"`
	Status         string    `json:"status"`
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

// ValidateStatus checks if the given status is valid
func ValidateStatus(status string) bool {
	return status == StatusUnread || status == StatusReading || status == StatusCompleted || status == StatusArchived
}
