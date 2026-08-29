package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "github.com/andrew-craig/cairn-reader/pkg/errors"
	"github.com/andrew-craig/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// CreateUser creates a new user with email and password
	CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error)

	// CreateMobileUser creates a new mobile-only user with Expo device ID
	CreateMobileUser(ctx context.Context, expoDeviceID string) (*models.User, error)

	// GetUserByID retrieves a user by their ID
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)

	// GetUserByExpoDeviceID retrieves a user by their Expo device ID
	GetUserByExpoDeviceID(ctx context.Context, expoDeviceID string) (*models.User, error)

	// UpdateUser updates a user's profile data (email only for now)
	UpdateUser(ctx context.Context, id uuid.UUID, email *string) (*models.User, error)

	// UpgradeAccount adds email and password to a mobile-only account
	UpgradeAccount(ctx context.Context, id uuid.UUID, email, passwordHash string) (*models.User, error)

	// UpdatePasswordHash updates a user's password hash
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) (*models.User, error)

	// DeleteUser deletes a user account
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// CreateEmailVerificationToken stores a hashed email verification token for a user.
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error

	// ConsumeEmailVerificationToken looks up a token by its hash, verifies it has not
	// expired, marks the owning user's email as verified, and deletes the token.
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*models.User, error)

	// UpdateLastLoginAt updates the last login timestamp for a user
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error

	// RecordFailedLogin increments failed_login_attempts, sets last_failed_login_at,
	// and locks the account if the threshold is reached.
	RecordFailedLogin(ctx context.Context, id uuid.UUID, lockoutThreshold int, lockoutDuration time.Duration) error

	// ResetFailedLogins clears failed login tracking on a successful login.
	ResetFailedLogins(ctx context.Context, id uuid.UUID) error
}

// userRepository is the concrete implementation of UserRepository
type userRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser creates a new user with email and password
func (r *userRepository) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	if email == "" || passwordHash == "" {
		return nil, apperrors.ErrInvalidUserData
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        &email,
		PasswordHash: &passwordHash,
		ExpoDeviceID: nil,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		LastLoginAt:  nil,
	}

	query := `
		INSERT INTO users (id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		          failed_login_attempts, locked_until, last_failed_login_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.ExpoDeviceID,
		user.EmailVerified,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLoginAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// CreateMobileUser creates a new mobile-only user with Expo device ID
func (r *userRepository) CreateMobileUser(ctx context.Context, expoDeviceID string) (*models.User, error) {
	if expoDeviceID == "" {
		return nil, apperrors.ErrInvalidUserData
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        nil,
		PasswordHash: nil,
		ExpoDeviceID: &expoDeviceID,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
		LastLoginAt:  nil,
	}

	query := `
		INSERT INTO users (id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		          failed_login_attempts, locked_until, last_failed_login_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.ExpoDeviceID,
		user.EmailVerified,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLoginAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create mobile user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		       failed_login_attempts, locked_until, last_failed_login_at
		FROM users
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, apperrors.ErrInvalidUserData
	}

	user := &models.User{}

	query := `
		SELECT id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		       failed_login_attempts, locked_until, last_failed_login_at
		FROM users
		WHERE email = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByExpoDeviceID retrieves a user by their Expo device ID
func (r *userRepository) GetUserByExpoDeviceID(ctx context.Context, expoDeviceID string) (*models.User, error) {
	if expoDeviceID == "" {
		return nil, apperrors.ErrInvalidUserData
	}

	user := &models.User{}

	query := `
		SELECT id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		       failed_login_attempts, locked_until, last_failed_login_at
		FROM users
		WHERE expo_device_id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, expoDeviceID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by expo device ID: %w", err)
	}

	return user, nil
}

// UpdateUser updates a user's profile data
// Currently only supports updating email, but can be extended for other fields
func (r *userRepository) UpdateUser(ctx context.Context, id uuid.UUID, email *string) (*models.User, error) {
	// First, verify the user exists
	user, err := r.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update the email if provided
	if email != nil {
		user.Email = email
	}

	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users
		SET email = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		          failed_login_attempts, locked_until, last_failed_login_at
	`

	err = r.db.Pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.UpdatedAt,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// UpgradeAccount adds email and password to a mobile-only account
func (r *userRepository) UpgradeAccount(ctx context.Context, id uuid.UUID, email, passwordHash string) (*models.User, error) {
	if email == "" || passwordHash == "" {
		return nil, apperrors.ErrInvalidUserData
	}

	// First, verify the user exists and is mobile-only
	user, err := r.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify this is a mobile-only account
	if !user.IsMobileOnly() {
		return nil, fmt.Errorf("account is not mobile-only: %w", apperrors.ErrInvalidUserData)
	}

	user.Email = &email
	user.PasswordHash = &passwordHash
	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users
		SET email = $1, password_hash = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		          failed_login_attempts, locked_until, last_failed_login_at
	`

	err = r.db.Pool.QueryRow(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.UpdatedAt,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation on email
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, apperrors.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to upgrade account: %w", err)
	}

	return user, nil
}

// UpdatePasswordHash updates a user's password hash
func (r *userRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) (*models.User, error) {
	if passwordHash == "" {
		return nil, apperrors.ErrInvalidUserData
	}

	now := time.Now().UTC()

	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		          failed_login_attempts, locked_until, last_failed_login_at
	`

	user := &models.User{}
	err := r.db.Pool.QueryRow(
		ctx,
		query,
		passwordHash,
		now,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastFailedLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	return user, nil
}

// DeleteUser deletes a user account
func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

// UpdateLastLoginAt updates the last login timestamp for a user
func (r *userRepository) UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE users
		SET last_login_at = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.Pool.Exec(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

// RecordFailedLogin increments the failed login counter and sets last_failed_login_at.
// If the new count reaches lockoutThreshold, the account is locked for lockoutDuration.
func (r *userRepository) RecordFailedLogin(ctx context.Context, id uuid.UUID, lockoutThreshold int, lockoutDuration time.Duration) error {
	now := time.Now().UTC()
	lockUntil := now.Add(lockoutDuration)

	// Increment the counter. If the new value hits the threshold, also set locked_until.
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    last_failed_login_at  = $1,
		    locked_until = CASE
		        WHEN failed_login_attempts + 1 >= $2 THEN $3
		        ELSE locked_until
		    END,
		    updated_at = $1
		WHERE id = $4
	`

	result, err := r.db.Pool.Exec(ctx, query, now, lockoutThreshold, lockUntil, id)
	if err != nil {
		return fmt.Errorf("failed to record failed login: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

// ResetFailedLogins clears failed login tracking after a successful login.
func (r *userRepository) ResetFailedLogins(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE users
		SET failed_login_attempts = 0,
		    locked_until           = NULL,
		    last_failed_login_at   = NULL,
		    updated_at             = $1
		WHERE id = $2
	`

	result, err := r.db.Pool.Exec(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to reset failed logins: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

// CreateEmailVerificationToken stores a hashed email verification token.
func (r *userRepository) CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM email_verification_tokens WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to clear old verification tokens: %w", err)
	}

	_, err = r.db.Pool.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}
	return nil
}

// ConsumeEmailVerificationToken validates the token, marks the user's email as verified,
// and deletes the token in a single transaction.
func (r *userRepository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*models.User, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID uuid.UUID
	var expiresAt time.Time
	err = tx.QueryRow(ctx,
		`DELETE FROM email_verification_tokens WHERE token_hash = $1 RETURNING user_id, expires_at`,
		tokenHash,
	).Scan(&userID, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to look up verification token: %w", err)
	}

	if time.Now().UTC().After(expiresAt) {
		return nil, apperrors.ErrUserNotFound
	}

	user := &models.User{}
	err = tx.QueryRow(ctx,
		`UPDATE users SET email_verified = TRUE, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, email, password_hash, expo_device_id, email_verified, created_at, updated_at, last_login_at,
		           failed_login_attempts, locked_until, last_failed_login_at`,
		userID,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.ExpoDeviceID,
		&user.EmailVerified, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.FailedLoginAttempts, &user.LockedUntil, &user.LastFailedLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to mark email verified: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return user, nil
}
