package models

import "time"

// RecommendationEvent represents a record of an article being recommended to a user
// This tracks individual recommendation events for analytics and avoiding repetition
type RecommendationEvent struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	ArticleID     string    `json:"article_id"`
	RecommendedAt time.Time `json:"recommended_at"`
}
