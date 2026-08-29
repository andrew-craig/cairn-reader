// Package services provides the business logic layer for the user service.
// It contains the AuthService for authentication operations (registration, login, token management)
// and the UserService for user profile management (get, update, upgrade, delete).
// Services orchestrate between handlers (HTTP layer) and repositories (database layer).
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
)

var (
	// ErrInvalidCredentials is returned when login credentials are incorrect
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrRefreshTokenExpired is returned when a refresh token has expired
	ErrRefreshTokenExpired = errors.New("refresh token expired")

	// ErrAccountExists is returned when attempting to register with an existing email or device ID
	ErrAccountExists = errors.New("account already exists")

	// ErrHybridAccountDeviceLogin is returned when attempting device login on a hybrid account
	ErrHybridAccountDeviceLogin = errors.New("device login not allowed for hybrid accounts")

	// ErrWeakPassword is returned when password doesn't meet strength requirements
	ErrWeakPassword = errors.New("password does not meet strength requirements")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrAccountLocked is returned when the account has been temporarily locked
	ErrAccountLocked = errors.New("account temporarily locked")

	// ErrInvalidVerificationToken is returned when a verification token is invalid or expired
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")

	// ErrEmailAlreadyVerified is returned when attempting to verify an already-verified email
	ErrEmailAlreadyVerified = errors.New("email already verified")
)

const (
	// defaultLockoutThreshold is the number of consecutive failures before locking
	defaultLockoutThreshold = 5
	// defaultLockoutDuration is how long an account remains locked
	defaultLockoutDuration = 15 * time.Minute
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
	lockoutThreshold    int
	lockoutDuration     time.Duration
}

// AuthServiceConfig holds configuration for the authentication service
type AuthServiceConfig struct {
	UserRepo            database.UserRepository
	RefreshTokenService *auth.RefreshTokenService
	JWTManager          *auth.JWTManager
	PasswordHasher      *auth.PasswordHasher
	PasswordMinLength   int           // Default: 8
	RequireComplexity   bool          // Default: true
	LockoutThreshold    int           // Default: 5 — consecutive failures before lockout
	LockoutDuration     time.Duration // Default: 15 minutes
}

// NewAuthService creates a new authentication service
func NewAuthService(config AuthServiceConfig) AuthService {
	// Set defaults
	if config.PasswordMinLength == 0 {
		config.PasswordMinLength = 8
	}
	if config.LockoutThreshold == 0 {
		config.LockoutThreshold = defaultLockoutThreshold
	}
	if config.LockoutDuration == 0 {
		config.LockoutDuration = defaultLockoutDuration
	}

	return &authService{
		userRepo:            config.UserRepo,
		refreshTokenService: config.RefreshTokenService,
		jwtManager:          config.JWTManager,
		passwordHasher:      config.PasswordHasher,
		passwordMinLength:   config.PasswordMinLength,
		requireComplexity:   config.RequireComplexity,
		lockoutThreshold:    config.LockoutThreshold,
		lockoutDuration:     config.LockoutDuration,
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
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			return nil, ErrAccountExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	auditEvent("account_created",
		slog.String("user_id", user.ID.String()),
		slog.String("auth_method", "email"),
	)

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
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			return nil, ErrAccountExists
		}
		return nil, fmt.Errorf("failed to create mobile user: %w", err)
	}

	auditEvent("account_created",
		slog.String("user_id", user.ID.String()),
		slog.String("auth_method", "mobile"),
		slog.String("ip", ipAddress),
	)

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
		if errors.Is(err, apperrors.ErrUserNotFound) {
			auditEvent("login_failure",
				slog.String("auth_method", "email"),
				slog.String("reason", "user_not_found"),
				slog.String("ip", ipAddress),
				slog.String("user_agent", deviceInfo),
			)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Check account lockout before verifying password
	if user.IsLocked() {
		minutesRemaining := int(math.Ceil(time.Until(*user.LockedUntil).Minutes()))
		return nil, fmt.Errorf("%w: try again after %d minute(s)", ErrAccountLocked, minutesRemaining)
	}

	// Verify user can login with email
	if !user.CanLoginWithEmail() {
		auditEvent("login_failure",
			slog.String("user_id", user.ID.String()),
			slog.String("auth_method", "email"),
			slog.String("reason", "method_not_allowed"),
			slog.String("ip", ipAddress),
		)
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := s.passwordHasher.ComparePassword(*user.PasswordHash, password); err != nil {
		auditEvent("login_failure",
			slog.String("user_id", user.ID.String()),
			slog.String("auth_method", "email"),
			slog.String("reason", "wrong_password"),
			slog.String("ip", ipAddress),
			slog.String("user_agent", deviceInfo),
		)
		// Record failed attempt; ignore secondary error so primary error is returned
		if recordErr := s.userRepo.RecordFailedLogin(ctx, user.ID, s.lockoutThreshold, s.lockoutDuration); recordErr != nil {
			slog.Warn("failed to record failed login attempt",
				slog.String("user_id", user.ID.String()),
				slog.Any("error", recordErr),
			)
		}
		return nil, ErrInvalidCredentials
	}

	// Successful login: reset failed-login tracking
	if err := s.userRepo.ResetFailedLogins(ctx, user.ID); err != nil {
		slog.Warn("failed to reset failed login counter",
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err),
		)
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		// Log error but don't fail login
		slog.Warn("failed to update last login timestamp",
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err),
		)
	}

	auditEvent("login_success",
		slog.String("user_id", user.ID.String()),
		slog.String("auth_method", "email"),
		slog.String("ip", ipAddress),
		slog.String("user_agent", deviceInfo),
	)

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
		if errors.Is(err, apperrors.ErrUserNotFound) {
			auditEvent("login_failure",
				slog.String("auth_method", "mobile"),
				slog.String("reason", "user_not_found"),
				slog.String("ip", ipAddress),
			)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Check account lockout before proceeding
	if user.IsLocked() {
		minutesRemaining := int(math.Ceil(time.Until(*user.LockedUntil).Minutes()))
		return nil, fmt.Errorf("%w: try again after %d minute(s)", ErrAccountLocked, minutesRemaining)
	}

	// Verify user can login with device (must be mobile-only, not hybrid)
	if !user.CanLoginWithDevice() {
		auditEvent("login_failure",
			slog.String("user_id", user.ID.String()),
			slog.String("auth_method", "mobile"),
			slog.String("reason", "hybrid_account_device_login"),
			slog.String("ip", ipAddress),
		)
		return nil, ErrHybridAccountDeviceLogin
	}

	// Successful login: reset failed-login tracking
	if err := s.userRepo.ResetFailedLogins(ctx, user.ID); err != nil {
		slog.Warn("failed to reset failed login counter",
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err),
		)
	}

	// Update last login timestamp
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID); err != nil {
		// Log error but don't fail login
		slog.Warn("failed to update last login timestamp",
			slog.String("user_id", user.ID.String()),
			slog.Any("error", err),
		)
	}

	auditEvent("login_success",
		slog.String("user_id", user.ID.String()),
		slog.String("auth_method", "mobile"),
		slog.String("ip", ipAddress),
	)

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
		slog.Warn("refresh token validation: empty token provided")
		return nil, ErrInvalidInput
	}

	slog.Debug("refresh token validation: starting")

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
			slog.Warn("refresh token validation: token reuse detected",
				slog.String("error", err.Error()),
			)
			auditEvent("token_reuse_detected",
				slog.String("ip", ipAddress),
				slog.String("user_agent", deviceInfo),
			)
			return nil, fmt.Errorf("token reuse detected: %w", err)
		}
		if errors.Is(err, auth.ErrRefreshTokenNotFound) {
			slog.Warn("refresh token validation: token not found")
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, auth.ErrTokenExpired) {
			slog.Warn("refresh token validation: token expired",
				slog.String("error", err.Error()),
			)
			return nil, ErrRefreshTokenExpired
		}
		slog.Error("refresh token validation: unexpected error",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	slog.Debug("refresh token validation: token validated successfully",
		slog.String("user_id", userID.String()),
	)

	// Retrieve user
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		slog.Error("refresh token validation: failed to retrieve user",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		slog.Error("refresh token validation: failed to generate access token",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	slog.Debug("refresh token validation: completed successfully",
		slog.String("user_id", userID.String()),
	)

	auditEvent("token_refreshed",
		slog.String("user_id", userID.String()),
		slog.String("ip", ipAddress),
	)

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
