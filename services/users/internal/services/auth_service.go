// Package services provides the business logic layer for the user service.
// It contains the AuthService for authentication operations (registration, login, token management)
// and the UserService for user profile management (get, update, upgrade, delete).
// Services orchestrate between handlers (HTTP layer) and repositories (database layer).
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/andrew-craig/cairn-core/user-service/internal/auth"
	"github.com/andrew-craig/cairn-core/user-service/internal/database"
	"github.com/andrew-craig/cairn-core/user-service/internal/models"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials is returned when login credentials are incorrect
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrAccountExists is returned when attempting to register with an existing email or device ID
	ErrAccountExists = errors.New("account already exists")

	// ErrHybridAccountDeviceLogin is returned when attempting device login on a hybrid account
	ErrHybridAccountDeviceLogin = errors.New("device login not allowed for hybrid accounts")

	// ErrWeakPassword is returned when password doesn't meet strength requirements
	ErrWeakPassword = errors.New("password does not meet strength requirements")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Register creates a new user account with email and password
	Register(ctx context.Context, email, password string) (*AuthResponse, error)

	// RegisterMobile creates a new mobile-only account with Expo device ID
	RegisterMobile(ctx context.Context, expoDeviceID, deviceInfo, ipAddress string) (*AuthResponse, error)

	// Login authenticates a user with email and password
	Login(ctx context.Context, email, password, deviceInfo, ipAddress string) (*AuthResponse, error)

	// LoginMobile authenticates a user with Expo device ID
	LoginMobile(ctx context.Context, expoDeviceID, deviceInfo, ipAddress string) (*AuthResponse, error)

	// RefreshAccessToken validates a refresh token and issues a new access token
	RefreshAccessToken(ctx context.Context, refreshToken, deviceInfo, ipAddress string) (*AuthResponse, error)

	// Logout revokes a specific refresh token
	Logout(ctx context.Context, refreshToken string) error

	// LogoutAll revokes all refresh tokens for a user
	LogoutAll(ctx context.Context, userID uuid.UUID) error
}

// AuthResponse contains the response data for authentication operations
type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"` // Access token expiration in seconds
}

// authService is the concrete implementation of AuthService
type authService struct {
	userRepo            database.UserRepository
	refreshTokenService *auth.RefreshTokenService
	jwtManager          *auth.JWTManager
	passwordHasher      *auth.PasswordHasher
	passwordMinLength   int
	requireComplexity   bool
}

// AuthServiceConfig holds configuration for the authentication service
type AuthServiceConfig struct {
	UserRepo            database.UserRepository
	RefreshTokenService *auth.RefreshTokenService
	JWTManager          *auth.JWTManager
	PasswordHasher      *auth.PasswordHasher
	PasswordMinLength   int  // Default: 8
	RequireComplexity   bool // Default: true
}

// NewAuthService creates a new authentication service
func NewAuthService(config AuthServiceConfig) AuthService {
	// Set defaults
	if config.PasswordMinLength == 0 {
		config.PasswordMinLength = 8
	}

	return &authService{
		userRepo:            config.UserRepo,
		refreshTokenService: config.RefreshTokenService,
		jwtManager:          config.JWTManager,
		passwordHasher:      config.PasswordHasher,
		passwordMinLength:   config.PasswordMinLength,
		requireComplexity:   config.RequireComplexity,
	}
}

// Register creates a new user account with email and password
func (s *authService) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
	// Validate input
	if email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(password, s.passwordMinLength, s.requireComplexity); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWeakPassword, err)
	}

	// Hash password
	passwordHash, err := s.passwordHasher.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userRepo.CreateUser(ctx, email, passwordHash)
	if err != nil {
		if errors.Is(err, database.ErrUserAlreadyExists) {
			return nil, ErrAccountExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user, nil, nil)
}

// RegisterMobile creates a new mobile-only account with Expo device ID
func (s *authService) RegisterMobile(ctx context.Context, expoDeviceID, deviceInfo, ipAddress string) (*AuthResponse, error) {
	// Validate input
	if expoDeviceID == "" {
		return nil, ErrInvalidInput
	}

	// Create mobile user
	user, err := s.userRepo.CreateMobileUser(ctx, expoDeviceID)
	if err != nil {
		if errors.Is(err, database.ErrUserAlreadyExists) {
			return nil, ErrAccountExists
		}
		return nil, fmt.Errorf("failed to create mobile user: %w", err)
	}

	// Prepare device info and IP for token creation
	var deviceInfoPtr, ipAddressPtr *string
	if deviceInfo != "" {
		deviceInfoPtr = &deviceInfo
	}
	if ipAddress != "" {
		ipAddressPtr = &ipAddress
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user, deviceInfoPtr, ipAddressPtr)
}

// Login authenticates a user with email and password
func (s *authService) Login(ctx context.Context, email, password, deviceInfo, ipAddress string) (*AuthResponse, error) {
	// Validate input
	if email == "" || password == "" {
		return nil, ErrInvalidInput
	}

	// Retrieve user by email
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify user can login with email
	if !user.CanLoginWithEmail() {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := s.passwordHasher.ComparePassword(*user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("warning: failed to update last login timestamp: %v\n", err)
	}

	// Prepare device info and IP for token creation
	var deviceInfoPtr, ipAddressPtr *string
	if deviceInfo != "" {
		deviceInfoPtr = &deviceInfo
	}
	if ipAddress != "" {
		ipAddressPtr = &ipAddress
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user, deviceInfoPtr, ipAddressPtr)
}

// LoginMobile authenticates a user with Expo device ID
func (s *authService) LoginMobile(ctx context.Context, expoDeviceID, deviceInfo, ipAddress string) (*AuthResponse, error) {
	// Validate input
	if expoDeviceID == "" {
		return nil, ErrInvalidInput
	}

	// Retrieve user by Expo device ID
	user, err := s.userRepo.GetUserByExpoDeviceID(ctx, expoDeviceID)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify user can login with device (must be mobile-only, not hybrid)
	if !user.CanLoginWithDevice() {
		return nil, ErrHybridAccountDeviceLogin
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("warning: failed to update last login timestamp: %v\n", err)
	}

	// Prepare device info and IP for token creation
	var deviceInfoPtr, ipAddressPtr *string
	if deviceInfo != "" {
		deviceInfoPtr = &deviceInfo
	}
	if ipAddress != "" {
		ipAddressPtr = &ipAddress
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user, deviceInfoPtr, ipAddressPtr)
}

// RefreshAccessToken validates a refresh token and issues a new access token
func (s *authService) RefreshAccessToken(ctx context.Context, refreshToken, deviceInfo, ipAddress string) (*AuthResponse, error) {
	// Validate input
	if refreshToken == "" {
		return nil, ErrInvalidInput
	}

	// Prepare device info and IP for token creation
	var deviceInfoPtr, ipAddressPtr *string
	if deviceInfo != "" {
		deviceInfoPtr = &deviceInfo
	}
	if ipAddress != "" {
		ipAddressPtr = &ipAddress
	}

	// Validate and rotate refresh token
	newRefreshToken, userID, err := s.refreshTokenService.ValidateAndRotateToken(
		ctx,
		refreshToken,
		deviceInfoPtr,
		ipAddressPtr,
	)
	if err != nil {
		if errors.Is(err, auth.ErrTokenReused) {
			return nil, fmt.Errorf("token reuse detected: %w", err)
		}
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, database.ErrTokenExpired) {
			return nil, fmt.Errorf("refresh token expired: %w", err)
		}
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	// Retrieve user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.jwtManager.GetTokenExpiry().Seconds()),
	}, nil
}

// Logout revokes a specific refresh token
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	// Validate input
	if refreshToken == "" {
		return ErrInvalidInput
	}

	// Revoke the token
	if err := s.refreshTokenService.RevokeToken(ctx, refreshToken); err != nil {
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			// Token already revoked or doesn't exist - this is fine
			return nil
		}
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// LogoutAll revokes all refresh tokens for a user
func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// Revoke all user tokens
	if err := s.refreshTokenService.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// generateAuthResponse creates an AuthResponse with access and refresh tokens
func (s *authService) generateAuthResponse(ctx context.Context, user *models.User, deviceInfo, ipAddress *string) (*AuthResponse, error) {
	// Generate JWT access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, _, err := s.refreshTokenService.CreateRefreshToken(
		ctx,
		user.ID,
		deviceInfo,
		ipAddress,
		nil, // New token family
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetTokenExpiry().Seconds()),
	}, nil
}
