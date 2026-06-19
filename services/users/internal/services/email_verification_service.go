package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
)

const (
	// verificationTokenBytes is the number of random bytes used for the token.
	verificationTokenBytes = 32
	// verificationTokenExpiry is how long a verification token is valid.
	verificationTokenExpiry = 24 * time.Hour
)

// EmailVerificationService handles email verification token generation and validation.
type EmailVerificationService interface {
	// SendVerificationEmail generates a verification token for the given user,
	// stores it in the database, and logs the verification URL (actual email
	// sending is not yet implemented and can be wired up later).
	SendVerificationEmail(ctx context.Context, userID uuid.UUID) error

	// VerifyEmail consumes a verification token and marks the user's email as verified.
	// Returns the updated user on success.
	VerifyEmail(ctx context.Context, token string) (*models.User, error)
}

// emailVerificationService is the concrete implementation.
type emailVerificationService struct {
	userRepo database.UserRepository
	baseURL  string // e.g. "https://api.cairn.example.com"
}

// NewEmailVerificationService creates a new EmailVerificationService.
// baseURL is used when logging the verification URL.
func NewEmailVerificationService(userRepo database.UserRepository, baseURL string) EmailVerificationService {
	return &emailVerificationService{
		userRepo: userRepo,
		baseURL:  baseURL,
	}
}

// SendVerificationEmail generates a secure token, stores its hash, and logs the URL.
func (s *emailVerificationService) SendVerificationEmail(ctx context.Context, userID uuid.UUID) error {
	// Check user exists and has an email
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}
	if user.Email == nil {
		return ErrInvalidInput
	}
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	// Generate cryptographically random token
	rawBytes := make([]byte, verificationTokenBytes)
	if _, err := rand.Read(rawBytes); err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}
	token := hex.EncodeToString(rawBytes)

	// Hash before storage
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().UTC().Add(verificationTokenExpiry)

	if err := s.userRepo.CreateEmailVerificationToken(ctx, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}

	// Log the verification URL (email sending is not implemented yet)
	verificationURL := fmt.Sprintf("%s/api/v1/auth/verify-email?token=%s", s.baseURL, token)
	slog.Info("email verification token generated",
		slog.String("user_id", userID.String()),
		slog.String("email", *user.Email),
		slog.String("verification_url", verificationURL),
		slog.Time("expires_at", expiresAt),
	)

	return nil
}

// VerifyEmail hashes the given token and delegates to the repository to atomically
// validate and consume it, marking the user's email as verified.
func (s *emailVerificationService) VerifyEmail(ctx context.Context, token string) (*models.User, error) {
	if token == "" {
		return nil, ErrInvalidInput
	}

	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	user, err := s.userRepo.ConsumeEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, ErrInvalidVerificationToken
		}
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	return user, nil
}
