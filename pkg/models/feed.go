package models

import "time"

// Feed represents an RSS feed source managed by the fetcher
type Feed struct {
	ID                  int        `json:"id"`
	URL                 string     `json:"url"`
	Title               string     `json:"title,omitempty"`
	Description         string     `json:"description,omitempty"`
	LastFetchedAt       *time.Time `json:"last_fetched_at,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	Enabled             bool       `json:"enabled"`
	ETag                string     `json:"etag,omitempty"`
	LastModified        string     `json:"last_modified,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// FetchHistory represents a record of a feed fetch attempt
type FetchHistory struct {
	ID               int        `json:"id"`
	FeedID           int        `json:"feed_id"`
	FetchStartedAt   time.Time  `json:"fetch_started_at"`
	FetchCompletedAt *time.Time `json:"fetch_completed_at,omitempty"`
	Success          bool       `json:"success"`
	ArticlesFound    int        `json:"articles_found"`
	ArticlesSent     int        `json:"articles_sent"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
