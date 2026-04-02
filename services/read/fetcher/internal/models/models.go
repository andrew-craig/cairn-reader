package models

import (
	"time"

	"github.com/google/uuid"
)

// PollingTier represents the polling frequency tier for a feed
type PollingTier string

const (
	PollingTierActive   PollingTier = "active"   // Poll every 1 hour
	PollingTierModerate PollingTier = "moderate" // Poll every 6 hours
	PollingTierQuiet    PollingTier = "quiet"    // Poll every 24 hours
)

// FeedStatus represents the status of a feed
type FeedStatus string

const (
	FeedStatusActive   FeedStatus = "active"
	FeedStatusDisabled FeedStatus = "disabled"
)

// ProcessingStatus represents the processing status of a feed item
type ProcessingStatus string

const (
	ProcessingStatusPending    ProcessingStatus = "pending"
	ProcessingStatusProcessing ProcessingStatus = "processing"
	ProcessingStatusCompleted  ProcessingStatus = "completed"
	ProcessingStatusFailed     ProcessingStatus = "failed"
)

// DeliveryStatus represents the delivery status of content in the outbox
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSending   DeliveryStatus = "sending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
)

// Feed represents an RSS/Atom feed
type Feed struct {
	ID      uuid.UUID `json:"id"`
	FeedURL string    `json:"feed_url"`

	// Feed metadata
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	SiteURL     *string `json:"site_url,omitempty"`

	// Polling management
	PollingTier     PollingTier `json:"polling_tier"`
	LastFetchedAt   *time.Time  `json:"last_fetched_at,omitempty"`
	LastPublishedAt *time.Time  `json:"last_published_at,omitempty"`
	NextPollAt      time.Time   `json:"next_poll_at"`

	// Status and error tracking
	Status               FeedStatus `json:"status"`
	ConsecutiveErrorDays int        `json:"consecutive_error_days"`
	LastErrorAt          *time.Time `json:"last_error_at,omitempty"`
	LastErrorMessage     *string    `json:"last_error_message,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FeedSubscription represents a user's subscription to a feed
type FeedSubscription struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	FeedID       uuid.UUID `json:"feed_id"`
	SubscribedAt time.Time `json:"subscribed_at"`
}

// FeedItem represents a single item/article from an RSS feed
type FeedItem struct {
	ID     uuid.UUID `json:"id"`
	FeedID uuid.UUID `json:"feed_id"`

	// Item identifiers
	ItemURL  string  `json:"item_url"`
	ItemGUID *string `json:"item_guid,omitempty"`

	// Processing status
	ProcessingStatus ProcessingStatus `json:"processing_status"`
	ContentHash      *string          `json:"content_hash,omitempty"`
	ContentServiceID *uuid.UUID       `json:"content_service_id,omitempty"`

	// Item metadata (raw from RSS)
	Title       *string    `json:"title,omitempty"`
	Author      *string    `json:"author,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Description *string    `json:"description,omitempty"`

	// Content update tracking
	HTTPLastModified *string    `json:"http_last_modified,omitempty"`
	HTTPETag         *string    `json:"http_etag,omitempty"`
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"`

	// Error tracking
	RetryCount int     `json:"retry_count"`
	LastError  *string `json:"last_error,omitempty"`

	// Timestamps
	DiscoveredAt time.Time  `json:"discovered_at"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty"`
}

// ContentOutbox represents a content item ready to be delivered to the Content Service
type ContentOutbox struct {
	ID         uuid.UUID `json:"id"`
	FeedItemID uuid.UUID `json:"feed_item_id"`

	// Processed content ready for delivery
	ContentPayload map[string]interface{} `json:"content_payload"` // JSONB data
	UserIDs        []uuid.UUID            `json:"user_ids"`

	// Delivery status tracking
	DeliveryStatus DeliveryStatus `json:"delivery_status"`
	RetryCount     int            `json:"retry_count"`
	MaxRetries     int            `json:"max_retries"`
	NextRetryAt    time.Time      `json:"next_retry_at"`
	LastError      *string        `json:"last_error,omitempty"`

	// Response tracking
	ContentServiceID *uuid.UUID `json:"content_service_id,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}
