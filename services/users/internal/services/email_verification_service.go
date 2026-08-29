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

	apperrors "github.com/andrew-craig/cairn-reader/pkg/errors"
	"github.com/andrew-craig/cairn-reader/services/users/internal/database"
	"github.com/andrew-craig/cairn-reader/services/users/internal/models"
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
	// SendVerificationEmail generates a verification token for the given user
	// and stores its hash (actual email sending is not yet implemented and
	// can be wired up later).
	SendVerificationEmail(ctx context.Context, userID uuid.UUID) error

	// VerifyEmail consumes a verification token and marks the user's email as verified.
	// Returns the updated user on success.
	VerifyEmail(ctx context.Context, token string) (*models.User, error)
}

// emailVerificationService is the concrete implementation.
type emailVerificationService struct {
	userRepo database.UserRepository
}

// NewEmailVerificationService creates a new EmailVerificationService.
func NewEmailVerificationService(userRepo database.UserRepository) EmailVerificationService {
	return &emailVerificationService{
		userRepo: userRepo,
	}
}

// SendVerificationEmail generates a secure token and stores its hash.
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

	// Token generated and stored (email sending is not implemented yet).
	// Never log the raw token, the assembled verification URL, or the
	// user's email address here - it would be a working account-takeover
	// exploit for anyone with log access.
	slog.Info("email verification token generated",
		slog.String("user_id", userID.String()),
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
