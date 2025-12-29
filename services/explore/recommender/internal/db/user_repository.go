package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// EnsureUserExists creates a user if they don't already exist
// This implements the auto-create user behavior required by Phase 3
func (r *UserRepository) EnsureUserExists(ctx context.Context, userID string) error {
	// Validate UUID format
	if _, err := uuid.Parse(userID); err != nil {
		return fmt.Errorf("invalid user ID format (must be UUID): %w", err)
	}

	query := `
		INSERT INTO users (id, created_at)
		VALUES ($1, NOW())
		ON CONFLICT (id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	return nil
}

// MarkArticleAsRead marks an article as read for a user
func (r *UserRepository) MarkArticleAsRead(ctx context.Context, userID, articleID string) error {
	// Ensure user exists first
	if err := r.EnsureUserExists(ctx, userID); err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	query := `
		INSERT INTO user_articles (user_id, article_id, read, read_at)
		VALUES ($1, $2, true, $3)
		ON CONFLICT (user_id, article_id) DO UPDATE SET
			read = true,
			read_at = $3
	`

	_, err := r.db.Exec(ctx, query, userID, articleID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark article as read: %w", err)
	}

	return nil
}

// GetReadArticleIDs returns the IDs of articles a user has read
func (r *UserRepository) GetReadArticleIDs(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT article_id
		FROM user_articles
		WHERE user_id = $1 AND read = true
	`

	rows, err := r.db.Query(ctx, query, userID)
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
