package models

import "time"

// User represents a user in the recommendation system
type User struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"` // External user identifier
	CreatedAt time.Time `json:"created_at"`
}
