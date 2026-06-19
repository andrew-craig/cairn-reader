package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account in the system.
// Users can be one of three types based on which fields are populated:
// - Mobile-only: expo_device_id is NOT NULL, email and password_hash are NULL
// - Email-only: email and password_hash are NOT NULL, expo_device_id is NULL
// - Hybrid: all three fields (email, password_hash, expo_device_id) are NOT NULL
type User struct {
	// ID is the unique identifier for the user
	ID uuid.UUID `json:"id" db:"id"`

	// Email is the user's email address (nullable for mobile-only accounts)
	Email *string `json:"email,omitempty" db:"email" validate:"omitempty,email"`

	// PasswordHash is the bcrypt hash of the user's password (nullable for mobile-only accounts)
	// This field is never included in JSON responses
	PasswordHash *string `json:"-" db:"password_hash"`

	// ExpoDeviceID is the Expo Application Installation ID (nullable for email-only accounts)
	// This field is never included in JSON responses as it's treated as a credential
	ExpoDeviceID *string `json:"-" db:"expo_device_id"`

	// EmailVerified indicates whether the user has verified their email address
	EmailVerified bool `json:"email_verified" db:"email_verified"`

	// CreatedAt is the timestamp when the user account was created
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// UpdatedAt is the timestamp when the user account was last updated
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// LastLoginAt is the timestamp of the user's last successful login (nullable)
	LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`

	// FailedLoginAttempts is the number of consecutive failed login attempts
	FailedLoginAttempts int `json:"-" db:"failed_login_attempts"`

	// LockedUntil is the timestamp until which the account is locked (nullable)
	LockedUntil *time.Time `json:"-" db:"locked_until"`

	// LastFailedLoginAt is the timestamp of the most recent failed login attempt (nullable)
	LastFailedLoginAt *time.Time `json:"-" db:"last_failed_login_at"`
}

// IsLocked returns true if the account is currently locked
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}

// IsMobileOnly returns true if the user account is mobile-only
// (has expo_device_id but no email/password)
func (u *User) IsMobileOnly() bool {
	return u.ExpoDeviceID != nil && u.Email == nil && u.PasswordHash == nil
}

// IsEmailOnly returns true if the user account is email-only
// (has email/password but no expo_device_id)
func (u *User) IsEmailOnly() bool {
	return u.Email != nil && u.PasswordHash != nil && u.ExpoDeviceID == nil
}

// IsHybrid returns true if the user account is hybrid
// (has both email/password and expo_device_id)
func (u *User) IsHybrid() bool {
	return u.Email != nil && u.PasswordHash != nil && u.ExpoDeviceID != nil
}

// CanLoginWithEmail returns true if the user can authenticate with email/password
func (u *User) CanLoginWithEmail() bool {
	return u.Email != nil && u.PasswordHash != nil
}

// CanLoginWithDevice returns true if the user can authenticate with device ID
// Note: Hybrid accounts cannot login with device ID per requirements
func (u *User) CanLoginWithDevice() bool {
	return u.IsMobileOnly()
}

// AccountType returns a string representation of the account type
// for logging and debugging purposes
func (u *User) AccountType() string {
	if u.IsMobileOnly() {
		return "mobile-only"
	}
	if u.IsEmailOnly() {
		return "email-only"
	}
	if u.IsHybrid() {
		return "hybrid"
	}
	return "invalid"
}
