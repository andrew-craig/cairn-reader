package models

import (
	"time"

	"github.com/google/uuid"
)

// ProcessingStatus represents the processing status of a raw email
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

// EmailAddress represents a user's unique email address for receiving newsletters
type EmailAddress struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	LocalPart string    `json:"local_part"` // e.g. "k7m2x9pq"

	CreatedAt time.Time `json:"created_at"`
}

// EmailSender represents a distinct sender for grouping
type EmailSender struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	SenderEmail string    `json:"sender_email"`
	SenderName  *string   `json:"sender_name,omitempty"`

	EmailCount     int        `json:"email_count"`
	LastReceivedAt *time.Time `json:"last_received_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RawEmail represents an incoming email before processing
type RawEmail struct {
	ID       uuid.UUID  `json:"id"`
	UserID   uuid.UUID  `json:"user_id"`
	SenderID *uuid.UUID `json:"sender_id,omitempty"`

	// Raw email data from Cloudflare worker
	Recipient   string  `json:"recipient"`
	SenderEmail string  `json:"sender_email"`
	SenderName  *string `json:"sender_name,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	HTMLBody    *string `json:"html_body,omitempty"`
	TextBody    *string `json:"text_body,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`

	// Processing status
	ProcessingStatus ProcessingStatus `json:"processing_status"`
	ContentHash      *string          `json:"content_hash,omitempty"`
	RetryCount       int              `json:"retry_count"`
	LastError        *string          `json:"last_error,omitempty"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// ContentOutbox represents a content item ready to be delivered to the Content Service
type ContentOutbox struct {
	ID         uuid.UUID `json:"id"`
	RawEmailID uuid.UUID `json:"raw_email_id"`

	// Processed content ready for delivery
	ContentPayload map[string]interface{} `json:"content_payload"` // JSONB data
	UserID         uuid.UUID              `json:"user_id"`

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
