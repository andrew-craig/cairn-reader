package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUser_IsMobileOnly(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "mobile-only user",
			user: User{
				ID:           uuid.New(),
				ExpoDeviceID: stringPtr("device-123"),
				Email:        nil,
				PasswordHash: nil,
			},
			expected: true,
		},
		{
			name: "email-only user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: nil,
			},
			expected: false,
		},
		{
			name: "hybrid user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: stringPtr("device-123"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsMobileOnly(); got != tt.expected {
				t.Errorf("IsMobileOnly() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_IsEmailOnly(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "email-only user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: nil,
			},
			expected: true,
		},
		{
			name: "mobile-only user",
			user: User{
				ID:           uuid.New(),
				ExpoDeviceID: stringPtr("device-123"),
				Email:        nil,
				PasswordHash: nil,
			},
			expected: false,
		},
		{
			name: "hybrid user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: stringPtr("device-123"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsEmailOnly(); got != tt.expected {
				t.Errorf("IsEmailOnly() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_IsHybrid(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "hybrid user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: stringPtr("device-123"),
			},
			expected: true,
		},
		{
			name: "email-only user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: nil,
			},
			expected: false,
		},
		{
			name: "mobile-only user",
			user: User{
				ID:           uuid.New(),
				ExpoDeviceID: stringPtr("device-123"),
				Email:        nil,
				PasswordHash: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsHybrid(); got != tt.expected {
				t.Errorf("IsHybrid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_CanLoginWithDevice(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected bool
	}{
		{
			name: "mobile-only user can login with device",
			user: User{
				ID:           uuid.New(),
				ExpoDeviceID: stringPtr("device-123"),
				Email:        nil,
				PasswordHash: nil,
			},
			expected: true,
		},
		{
			name: "hybrid user cannot login with device",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: stringPtr("device-123"),
			},
			expected: false,
		},
		{
			name: "email-only user cannot login with device",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.CanLoginWithDevice(); got != tt.expected {
				t.Errorf("CanLoginWithDevice() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_AccountType(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected string
	}{
		{
			name: "mobile-only user",
			user: User{
				ID:           uuid.New(),
				ExpoDeviceID: stringPtr("device-123"),
				Email:        nil,
				PasswordHash: nil,
			},
			expected: "mobile-only",
		},
		{
			name: "email-only user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: nil,
			},
			expected: "email-only",
		},
		{
			name: "hybrid user",
			user: User{
				ID:           uuid.New(),
				Email:        stringPtr("user@example.com"),
				PasswordHash: stringPtr("hash"),
				ExpoDeviceID: stringPtr("device-123"),
			},
			expected: "hybrid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.AccountType(); got != tt.expected {
				t.Errorf("AccountType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    RefreshToken
		expected bool
	}{
		{
			name: "expired token",
			token: RefreshToken{
				ID:        uuid.New(),
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "valid token",
			token: RefreshToken{
				ID:        uuid.New(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRefreshToken_ShouldRotate(t *testing.T) {
	token := RefreshToken{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Per requirements, tokens should always rotate on use
	if !token.ShouldRotate() {
		t.Error("ShouldRotate() should always return true per requirements")
	}
}

// Helper function to create string pointers for test data
func stringPtr(s string) *string {
	return &s
}
