package services

import (
	"context"
	"errors"
	"fmt"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
)

var (
	// ErrUnauthorized is returned when a user attempts to access/modify another user's data
	ErrUnauthorized = errors.New("unauthorized access")

	// ErrNotMobileAccount is returned when attempting to upgrade a non-mobile account
	ErrNotMobileAccount = errors.New("account is not a mobile-only account")
)

// UserService defines the interface for user management operations
type UserService interface {
	// GetUser retrieves a user by ID with authorization check
	// The requestingUserID must match the requested ID unless the user has admin privileges
	GetUser(ctx context.Context, requestingUserID, targetUserID uuid.UUID) (*models.User, error)

	// UpdateUser updates a user's profile data with authorization check
	// The requestingUserID must match the target ID
	UpdateUser(ctx context.Context, requestingUserID, targetUserID uuid.UUID, email *string) (*models.User, error)

	// UpgradeAccount upgrades a mobile-only account to a hybrid account
	// The requestingUserID must match the target ID
	UpgradeAccount(ctx context.Context, requestingUserID, targetUserID uuid.UUID, email, password string) (*models.User, error)

	// DeleteUser deletes a user account with authorization check
	// The requestingUserID must match the target ID
	DeleteUser(ctx context.Context, requestingUserID, targetUserID uuid.UUID) error
}

// userService is the concrete implementation of UserService
type userService struct {
	userRepo          database.UserRepository
	refreshTokenRepo  database.RefreshTokenRepository
	passwordHasher    *auth.PasswordHasher
	passwordMinLength int
	requireComplexity bool
}

// UserServiceConfig holds configuration for the user service
type UserServiceConfig struct {
	UserRepo          database.UserRepository
	RefreshTokenRepo  database.RefreshTokenRepository
	PasswordHasher    *auth.PasswordHasher
	PasswordMinLength int  // Default: 8
	RequireComplexity bool // Default: true
}

// NewUserService creates a new user service
func NewUserService(config UserServiceConfig) UserService {
	// Set defaults
	if config.PasswordMinLength == 0 {
		config.PasswordMinLength = 8
	}

	return &userService{
		userRepo:          config.UserRepo,
		refreshTokenRepo:  config.RefreshTokenRepo,
		passwordHasher:    config.PasswordHasher,
		passwordMinLength: config.PasswordMinLength,
		requireComplexity: config.RequireComplexity,
	}
}

// GetUser retrieves a user by ID with authorization check
func (s *userService) GetUser(ctx context.Context, requestingUserID, targetUserID uuid.UUID) (*models.User, error) {
	// Authorization check: user can only access their own data
	if requestingUserID != targetUserID {
		return nil, ErrUnauthorized
	}

	// Retrieve user
	user, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user's profile data with authorization check
func (s *userService) UpdateUser(ctx context.Context, requestingUserID, targetUserID uuid.UUID, email *string) (*models.User, error) {
	// Authorization check: user can only update their own data
	if requestingUserID != targetUserID {
		return nil, ErrUnauthorized
	}

	// Validate input
	if email == nil {
		return nil, ErrInvalidInput
	}

	// Email uniqueness validation is handled by the repository layer (unique constraint)
	// Update user
	user, err := s.userRepo.UpdateUser(ctx, targetUserID, email)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			return nil, ErrAccountExists
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// UpgradeAccount upgrades a mobile-only account to a hybrid account
func (s *userService) UpgradeAccount(ctx context.Context, requestingUserID, targetUserID uuid.UUID, email, password string) (*models.User, error) {
	// Authorization check: user can only upgrade their own account
	if requestingUserID != targetUserID {
		return nil, ErrUnauthorized
	}

	// Validate input
	if email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(password, s.passwordMinLength, s.requireComplexity); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}

	// Verify the user exists and is mobile-only
	user, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify this is a mobile-only account
	if !user.IsMobileOnly() {
		return nil, ErrNotMobileAccount
	}

	// Hash password
	passwordHash, err := s.passwordHasher.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Email and device ID uniqueness validation is handled by the repository layer
	// Upgrade account
	upgradedUser, err := s.userRepo.UpgradeAccount(ctx, targetUserID, email, passwordHash)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			return nil, ErrAccountExists
		}
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to upgrade account: %w", err)
	}

	return upgradedUser, nil
}

// DeleteUser deletes a user account with authorization check
func (s *userService) DeleteUser(ctx context.Context, requestingUserID, targetUserID uuid.UUID) error {
	// Authorization check: user can only delete their own account
	if requestingUserID != targetUserID {
		return ErrUnauthorized
	}

	// First, revoke all refresh tokens for the user
	if err := s.refreshTokenRepo.RevokeAllUserTokens(ctx, targetUserID); err != nil {
		// Log error but continue with deletion
		fmt.Printf("warning: failed to revoke user tokens during deletion: %v\n", err)
	}

	// Delete user
	if err := s.userRepo.DeleteUser(ctx, targetUserID); err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return apperrors.ErrUserNotFound
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
