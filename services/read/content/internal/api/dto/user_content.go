package dto

import (
	"time"

	"github.com/google/uuid"
)

// AddContentToUserRequest represents the request body for adding content to a user's list
// Supports two modes:
// 1. URL-based (new): Provide URL, optional Type and Title for auto-detection/routing
// 2. Content-ID-based (legacy): Provide ContentID for pre-created content
type AddContentToUserRequest struct {
	// URL-based submission (new flow)
	URL   *string `json:"url,omitempty"`   // URL to add (triggers detection/routing)
	Type  *string `json:"type,omitempty"`  // Optional type hint: "feed", "page", or "unknown"
	Title *string `json:"title,omitempty"` // Optional pre-detected title

	// Content-ID-based submission (legacy flow)
	ContentID *uuid.UUID `json:"content_id,omitempty"`

	// Common metadata (applies to both flows)
	Status         string `json:"status,omitempty"`
	ScrollPosition int    `json:"scroll_position,omitempty"`
	IsFavorite     bool   `json:"is_favorite,omitempty"`
}

// AddFeedResponse represents a successful feed subscription
type AddFeedResponse struct {
	Type         string              `json:"type"` // Always "feed"
	FeedID       string              `json:"feed_id"`
	Subscription FeedSubscriptionDTO `json:"subscription"`
}

// AddPageResponse represents a successful page addition
type AddPageResponse struct {
	Type    string               `json:"type"` // Always "page"
	Content *UserContentResponse `json:"content"`
}

// FeedSubscriptionDTO represents a feed subscription from Ingest RSS service
type FeedSubscriptionDTO struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	FeedID       string    `json:"feed_id"`
	FeedURL      string    `json:"feed_url"`
	Title        string    `json:"title"`
	SubscribedAt time.Time `json:"subscribed_at"`
}

// UpdateUserContentRequest represents the request body for updating user-content metadata
type UpdateUserContentRequest struct {
	Status         *string `json:"status,omitempty"`
	ScrollPosition *int    `json:"scroll_position,omitempty"`
	IsFavorite     *bool   `json:"is_favorite,omitempty"`
}

// UserContentResponse represents a user-content item in API responses
type UserContentResponse struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	ContentID      uuid.UUID       `json:"content_id"`
	Status         string          `json:"status"`
	ScrollPosition int             `json:"scroll_position"`
	IsFavorite     bool            `json:"is_favorite"`
	AddedAt        time.Time       `json:"added_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Content        *ContentResponse `json:"content,omitempty"`
}

// UserContentsListResponse represents a list of user contents with pagination
type UserContentsListResponse struct {
	Contents   []*UserContentResponse `json:"contents"`
	TotalCount int64                  `json:"total_count"`
	Limit      int                    `json:"limit"`
	Offset     int                    `json:"offset"`
	NextCursor *string                `json:"next_cursor,omitempty"`
}
