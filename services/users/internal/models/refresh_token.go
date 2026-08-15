package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token used for obtaining new access tokens.
// Refresh tokens are long-lived credentials that allow users to obtain new
// access tokens without re-authenticating. They support rotation and family
// tracking for security.
type RefreshToken struct {
	// ID is the unique identifier for the refresh token
	ID uuid.UUID `json:"id" db:"id"`

	// UserID is the ID of the user this token belongs to
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	// TokenHash is the SHA-256 hash of the refresh token
	// The actual token is never stored, only its hash
	TokenHash string `json:"-" db:"token_hash"`

	// ExpiresAt is when this refresh token expires
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`

	// CreatedAt is when this refresh token was created
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// LastUsedAt is when this refresh token was last used to obtain an access token
	LastUsedAt time.Time `json:"last_used_at" db:"last_used_at"`

	// DeviceInfo contains user agent or device information for security tracking
	DeviceInfo *string `json:"device_info,omitempty" db:"device_info"`

	// IPAddress is the IP address from which the token was created
	IPAddress *string `json:"ip_address,omitempty" db:"ip_address"`

	// TokenFamily is a UUID used to track token rotation chains for reuse detection
	// All tokens in a rotation chain share the same family ID
	TokenFamily *uuid.UUID `json:"token_family,omitempty" db:"token_family"`

	// RevokedAt is when this token was revoked (rotated away or logged out).
	// Nil means the token is still active. Revoked tokens are retained until
	// expiry (rather than deleted) so a replay of them can be detected as reuse.
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// IsExpired returns true if the refresh token has expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid returns true if the refresh token is not expired
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsExpired()
}

// TimeUntilExpiry returns the duration until the token expires
// Returns 0 if already expired
func (rt *RefreshToken) TimeUntilExpiry() time.Duration {
	if rt.IsExpired() {
		return 0
	}
	return time.Until(rt.ExpiresAt)
}

// ShouldRotate returns true if the token should be rotated
// Per requirements, tokens are rotated on each use
func (rt *RefreshToken) ShouldRotate() bool {
	// Always rotate on use per requirements
	return true
}
