package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-core/user-service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when attempting to create a user with an email or device ID that already exists
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidUserData is returned when user data is invalid
	ErrInvalidUserData = errors.New("invalid user data")
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

	// DeleteUser deletes a user account
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// UpdateLastLoginAt updates the last login timestamp for a user
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error
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
		return nil, ErrInvalidUserData
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
		INSERT INTO users (id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.ExpoDeviceID,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLoginAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// CreateMobileUser creates a new mobile-only user with Expo device ID
func (r *userRepository) CreateMobileUser(ctx context.Context, expoDeviceID string) (*models.User, error) {
	if expoDeviceID == "" {
		return nil, ErrInvalidUserData
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
		INSERT INTO users (id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.ExpoDeviceID,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLoginAt,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create mobile user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, ErrInvalidUserData
	}

	user := &models.User{}

	query := `
		SELECT id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByExpoDeviceID retrieves a user by their Expo device ID
func (r *userRepository) GetUserByExpoDeviceID(ctx context.Context, expoDeviceID string) (*models.User, error) {
	if expoDeviceID == "" {
		return nil, ErrInvalidUserData
	}

	user := &models.User{}

	query := `
		SELECT id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
		FROM users
		WHERE expo_device_id = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, expoDeviceID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.ExpoDeviceID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
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
		RETURNING id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
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
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// UpgradeAccount adds email and password to a mobile-only account
func (r *userRepository) UpgradeAccount(ctx context.Context, id uuid.UUID, email, passwordHash string) (*models.User, error) {
	if email == "" || passwordHash == "" {
		return nil, ErrInvalidUserData
	}

	// First, verify the user exists and is mobile-only
	user, err := r.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify this is a mobile-only account
	if !user.IsMobileOnly() {
		return nil, fmt.Errorf("account is not mobile-only: %w", ErrInvalidUserData)
	}

	user.Email = &email
	user.PasswordHash = &passwordHash
	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users
		SET email = $1, password_hash = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, email, password_hash, expo_device_id, created_at, updated_at, last_login_at
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
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)

	if err != nil {
		// Check for unique constraint violation on email
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to upgrade account: %w", err)
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
		return ErrUserNotFound
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
		return ErrUserNotFound
	}

	return nil
}
