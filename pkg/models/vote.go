package models

import "time"

// Vote represents a user's vote (upvote or downvote) on an article
type Vote struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	ArticleID string    `json:"article_id"`
	VoteType  string    `json:"vote_type"` // "upvote" or "downvote"
	CreatedAt time.Time `json:"created_at"`
}
