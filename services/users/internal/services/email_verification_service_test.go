package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"testing"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUserRepo is a minimal in-memory UserRepository for unit tests.
// Only the methods used by EmailVerificationService are implemented.
type mockUserRepo struct {
	users  map[uuid.UUID]*models.User
	tokens map[string]mockVerificationToken // token_hash -> token
}

type mockVerificationToken struct {
	userID    uuid.UUID
	expiresAt time.Time
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:  make(map[uuid.UUID]*models.User),
		tokens: make(map[string]mockVerificationToken),
	}
}

func (m *mockUserRepo) addUser(u *models.User) {
	m.users[u.ID] = u
}

// UserRepository interface implementation (only what EmailVerificationService uses)

func (m *mockUserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, apperrors.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	// Delete previous tokens for this user
	for k, v := range m.tokens {
		if v.userID == userID {
			delete(m.tokens, k)
		}
	}
	m.tokens[tokenHash] = mockVerificationToken{userID: userID, expiresAt: expiresAt}
	return nil
}

func (m *mockUserRepo) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*models.User, error) {
	tok, ok := m.tokens[tokenHash]
	if !ok {
		return nil, apperrors.ErrUserNotFound
	}
	delete(m.tokens, tokenHash)
	if time.Now().UTC().After(tok.expiresAt) {
		return nil, apperrors.ErrUserNotFound
	}
	u, ok := m.users[tok.userID]
	if !ok {
		return nil, apperrors.ErrUserNotFound
	}
	u.EmailVerified = true
	return u, nil
}

// Stub out the remaining interface methods (not used by EmailVerificationService)
func (m *mockUserRepo) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) CreateMobileUser(ctx context.Context, expoDeviceID string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) GetUserByExpoDeviceID(ctx context.Context, expoDeviceID string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) UpdateUser(ctx context.Context, id uuid.UUID, email *string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) UpgradeAccount(ctx context.Context, id uuid.UUID, email, passwordHash string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) (*models.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) RecordFailedLogin(ctx context.Context, id uuid.UUID, threshold int, duration time.Duration) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) ResetFailedLogins(ctx context.Context, id uuid.UUID) error {
	return errors.New("not implemented")
}

// helper
func emailPtr(s string) *string { return &s }

func TestSendVerificationEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("generates and stores token for unverified user", func(t *testing.T) {
		repo := newMockUserRepo()
		userID := uuid.New()
		email := "user@example.com"
		repo.addUser(&models.User{
			ID:            userID,
			Email:         emailPtr(email),
			EmailVerified: false,
		})

		svc := NewEmailVerificationService(repo)
		err := svc.SendVerificationEmail(ctx, userID)
		require.NoError(t, err)
		// A token should now be stored
		assert.Len(t, repo.tokens, 1)
	})

	t.Run("returns ErrInvalidInput for mobile-only user (no email)", func(t *testing.T) {
		repo := newMockUserRepo()
		userID := uuid.New()
		repo.addUser(&models.User{
			ID:    userID,
			Email: nil,
		})

		svc := NewEmailVerificationService(repo)
		err := svc.SendVerificationEmail(ctx, userID)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("returns ErrEmailAlreadyVerified for already verified email", func(t *testing.T) {
		repo := newMockUserRepo()
		userID := uuid.New()
		repo.addUser(&models.User{
			ID:            userID,
			Email:         emailPtr("user@example.com"),
			EmailVerified: true,
		})

		svc := NewEmailVerificationService(repo)
		err := svc.SendVerificationEmail(ctx, userID)
		assert.ErrorIs(t, err, ErrEmailAlreadyVerified)
	})

	t.Run("returns error for unknown user", func(t *testing.T) {
		repo := newMockUserRepo()
		svc := NewEmailVerificationService(repo)
		err := svc.SendVerificationEmail(ctx, uuid.New())
		assert.Error(t, err)
	})

	t.Run("replaces existing token on resend", func(t *testing.T) {
		repo := newMockUserRepo()
		userID := uuid.New()
		repo.addUser(&models.User{
			ID:    userID,
			Email: emailPtr("user@example.com"),
		})

		svc := NewEmailVerificationService(repo)
		require.NoError(t, svc.SendVerificationEmail(ctx, userID))
		require.NoError(t, svc.SendVerificationEmail(ctx, userID))
		// Should still be exactly one token
		assert.Len(t, repo.tokens, 1)
	})
}

// hexRunPattern matches any run of 16+ hex characters, long enough to catch
// a raw or partial verification token (64 hex chars) leaking into a log line,
// while not matching shorter incidental hex-looking substrings like UUID segments.
var hexRunPattern = regexp.MustCompile(`[0-9a-fA-F]{16,}`)

// TestSendVerificationEmail_DoesNotLogSecretMaterial guards against P2-C2/H2:
// the verification token, the assembled verification URL, and the user's
// plaintext email must never appear in the logs emitted by SendVerificationEmail.
func TestSendVerificationEmail_DoesNotLogSecretMaterial(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	prevDefault := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(prevDefault)

	repo := newMockUserRepo()
	userID := uuid.New()
	email := "verify-target@example.com"
	repo.addUser(&models.User{
		ID:            userID,
		Email:         emailPtr(email),
		EmailVerified: false,
	})

	svc := NewEmailVerificationService(repo)
	require.NoError(t, svc.SendVerificationEmail(context.Background(), userID))

	output := buf.String()
	assert.NotContains(t, output, email, "log output must not contain the user's plaintext email")
	assert.NotContains(t, output, "verify-email", "log output must not contain a verification URL")
	assert.NotContains(t, output, "token=", "log output must not contain a token query parameter")
	assert.False(t, hexRunPattern.MatchString(output), "log output must not contain a raw or partial token: %s", output)

	// Every emitted record must stick to the non-secret allowlist of fields.
	allowedKeys := map[string]bool{
		"time": true, "level": true, "msg": true,
		"user_id": true, "expires_at": true,
	}
	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	sawRecord := false
	for {
		var entry map[string]interface{}
		if err := decoder.Decode(&entry); err != nil {
			break
		}
		sawRecord = true
		for key := range entry {
			assert.True(t, allowedKeys[key], "unexpected log field %q may carry secret material: %v", key, entry)
		}
	}
	require.True(t, sawRecord, "expected SendVerificationEmail to emit a log record")
}

func TestVerifyEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully verifies email with valid token", func(t *testing.T) {
		repo := newMockUserRepo()
		userID := uuid.New()
		repo.addUser(&models.User{
			ID:            userID,
			Email:         emailPtr("user@example.com"),
			EmailVerified: false,
		})

		svc := NewEmailVerificationService(repo)
		require.NoError(t, svc.SendVerificationEmail(ctx, userID))

		// Extract the raw token by reversing the stored hash (we need the real token).
		// Since we don't have it directly, let's extract from the stored hash by
		// using the service end-to-end: store a known token manually.
		knownToken := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
		hash := sha256.Sum256([]byte(knownToken))
		tokenHash := hex.EncodeToString(hash[:])
		repo.tokens[tokenHash] = mockVerificationToken{
			userID:    userID,
			expiresAt: time.Now().Add(1 * time.Hour),
		}

		user, err := svc.VerifyEmail(ctx, knownToken)
		require.NoError(t, err)
		assert.True(t, user.EmailVerified)
		assert.Equal(t, userID, user.ID)
	})

	t.Run("returns ErrInvalidVerificationToken for unknown token", func(t *testing.T) {
		repo := newMockUserRepo()
		svc := NewEmailVerificationService(repo)

		_, err := svc.VerifyEmail(ctx, "nonexistent-token")
		assert.ErrorIs(t, err, ErrInvalidVerificationToken)
	})

	t.Run("returns ErrInvalidInput for empty token", func(t *testing.T) {
		repo := newMockUserRepo()
		svc := NewEmailVerificationService(repo)

		_, err := svc.VerifyEmail(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("returns ErrInvalidVerificationToken for expired token", func(t *testing.T) {
		repo := newMockUserRepo()
		userID := uuid.New()
		repo.addUser(&models.User{ID: userID, Email: emailPtr("user@example.com")})

		expiredToken := "expiredtoken"
		hash := sha256.Sum256([]byte(expiredToken))
		tokenHash := hex.EncodeToString(hash[:])
		// Store as already-expired
		repo.tokens[tokenHash] = mockVerificationToken{
			userID:    userID,
			expiresAt: time.Now().Add(-1 * time.Hour),
		}

		svc := NewEmailVerificationService(repo)
		_, err := svc.VerifyEmail(ctx, expiredToken)
		assert.ErrorIs(t, err, ErrInvalidVerificationToken)
	})
}
