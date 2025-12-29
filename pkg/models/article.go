package models

import "time"

// Article represents a single article from an RSS feed
type Article struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Author      string    `json:"author"`
	Published   time.Time `json:"published"`
	FeedURL     string    `json:"feed_url"`
	FeedTitle   string    `json:"feed_title"`
	Categories  []string  `json:"categories"`
	FeedID      *int      `json:"feed_id,omitempty"` // Optional: reference to feeds table
	Upvotes     int       `json:"upvotes"`
	Downvotes   int       `json:"downvotes"`
	Recommends  int       `json:"recommends"`
	Deleted     bool      `json:"deleted"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Recommendation represents a recommended article for a user
type Recommendation struct {
	UserID    string    `json:"user_id"`
	Articles  []Article `json:"articles"`
	Generated time.Time `json:"generated"`
}
