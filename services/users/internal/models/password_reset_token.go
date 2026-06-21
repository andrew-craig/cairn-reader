package models

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken represents a one-time password reset token.
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// IsExpired returns true if the token has passed its expiry time.
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsValid returns true if the token can be used to reset a password.
func (t *PasswordResetToken) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed()
}
