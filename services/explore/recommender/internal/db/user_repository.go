package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateOrGetUser creates a user if not exists, or returns the existing user's internal ID
// This implements the auto-create user behavior required by Phase 3
func (r *UserRepository) CreateOrGetUser(ctx context.Context, externalUserID string) (int, error) {
	query := `
		INSERT INTO users (user_id, created_at)
		VALUES ($1, NOW())
		ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(ctx, query, externalUserID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create or get user: %w", err)
	}

	return id, nil
}

// MarkArticleAsRead marks an article as read for a user
func (r *UserRepository) MarkArticleAsRead(ctx context.Context, externalUserID, articleID string) error {
	// First, get or create the user to get their internal ID
	userID, err := r.CreateOrGetUser(ctx, externalUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	query := `
		INSERT INTO user_articles (user_id, article_id, read, read_at)
		VALUES ($1, $2, true, $3)
		ON CONFLICT (user_id, article_id) DO UPDATE SET
			read = true,
			read_at = $3
	`

	_, err = r.db.ExecContext(ctx, query, userID, articleID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark article as read: %w", err)
	}

	return nil
}

// GetReadArticleIDs returns the IDs of articles a user has read
func (r *UserRepository) GetReadArticleIDs(ctx context.Context, externalUserID string) ([]string, error) {
	// First, get the user's internal ID
	userID, err := r.CreateOrGetUser(ctx, externalUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	query := `
		SELECT article_id
		FROM user_articles
		WHERE user_id = $1 AND read = true
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get read articles: %w", err)
	}
	defer rows.Close()

	articleIDs := make([]string, 0)
	for rows.Next() {
		var articleID string
		if err := rows.Scan(&articleID); err != nil {
			return nil, fmt.Errorf("failed to scan article ID: %w", err)
		}
		articleIDs = append(articleIDs, articleID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating read articles: %w", err)
	}

	return articleIDs, nil
}
