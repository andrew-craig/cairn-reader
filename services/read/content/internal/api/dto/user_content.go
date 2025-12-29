package dto

import (
	"time"

	"github.com/google/uuid"
)

// AddContentToUserRequest represents the request body for adding content to a user's list
type AddContentToUserRequest struct {
	ContentID      uuid.UUID `json:"content_id"`
	Status         string    `json:"status,omitempty"`
	ScrollPosition int       `json:"scroll_position,omitempty"`
	IsFavorite     bool      `json:"is_favorite,omitempty"`
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
