package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/models"
	"github.com/google/uuid"
)

// FeedSubscriptionRepository defines the interface for feed subscription operations
type FeedSubscriptionRepository interface {
	// Create creates a new feed subscription
	Create(ctx context.Context, subscription *models.FeedSubscription) error

	// GetByID retrieves a subscription by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.FeedSubscription, error)

	// GetByUserAndFeed retrieves a subscription for a specific user and feed
	GetByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) (*models.FeedSubscription, error)

	// GetByUserID retrieves all subscriptions for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.FeedSubscription, error)

	// GetByFeedID retrieves all subscriptions for a feed
	GetByFeedID(ctx context.Context, feedID uuid.UUID) ([]*models.FeedSubscription, error)

	// GetSubscriberIDs retrieves all user IDs subscribed to a feed
	GetSubscriberIDs(ctx context.Context, feedID uuid.UUID) ([]uuid.UUID, error)

	// CountByUserID counts the number of subscriptions for a user
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// Delete deletes a subscription
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserAndFeed deletes a subscription by user and feed
	DeleteByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) error
}

// feedSubscriptionRepository is the concrete implementation
type feedSubscriptionRepository struct {
	db *sql.DB
}

// NewFeedSubscriptionRepository creates a new FeedSubscriptionRepository
func NewFeedSubscriptionRepository(db *sql.DB) FeedSubscriptionRepository {
	return &feedSubscriptionRepository{db: db}
}

// Create creates a new feed subscription
func (r *feedSubscriptionRepository) Create(ctx context.Context, subscription *models.FeedSubscription) error {
	query := `
		INSERT INTO feed_subscriptions (id, user_id, feed_id, subscribed_at)
		VALUES ($1, $2, $3, $4)
	`

	if subscription.ID == uuid.Nil {
		subscription.ID = uuid.New()
	}
	if subscription.SubscribedAt.IsZero() {
		subscription.SubscribedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query,
		subscription.ID,
		subscription.UserID,
		subscription.FeedID,
		subscription.SubscribedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// GetByID retrieves a subscription by its ID
func (r *feedSubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FeedSubscription, error) {
	query := `
		SELECT id, user_id, feed_id, subscribed_at
		FROM feed_subscriptions
		WHERE id = $1
	`

	subscription := &models.FeedSubscription{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.FeedID,
		&subscription.SubscribedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by ID: %w", err)
	}

	return subscription, nil
}

// GetByUserAndFeed retrieves a subscription for a specific user and feed
func (r *feedSubscriptionRepository) GetByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) (*models.FeedSubscription, error) {
	query := `
		SELECT id, user_id, feed_id, subscribed_at
		FROM feed_subscriptions
		WHERE user_id = $1 AND feed_id = $2
	`

	subscription := &models.FeedSubscription{}
	err := r.db.QueryRowContext(ctx, query, userID, feedID).Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.FeedID,
		&subscription.SubscribedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by user and feed: %w", err)
	}

	return subscription, nil
}

// GetByUserID retrieves all subscriptions for a user
func (r *feedSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.FeedSubscription, error) {
	query := `
		SELECT id, user_id, feed_id, subscribed_at
		FROM feed_subscriptions
		WHERE user_id = $1
		ORDER BY subscribed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions by user: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]*models.FeedSubscription, 0)
	for rows.Next() {
		subscription := &models.FeedSubscription{}
		err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.FeedID,
			&subscription.SubscribedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscriptions: %w", err)
	}

	return subscriptions, nil
}

// GetByFeedID retrieves all subscriptions for a feed
func (r *feedSubscriptionRepository) GetByFeedID(ctx context.Context, feedID uuid.UUID) ([]*models.FeedSubscription, error) {
	query := `
		SELECT id, user_id, feed_id, subscribed_at
		FROM feed_subscriptions
		WHERE feed_id = $1
		ORDER BY subscribed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriptions by feed: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]*models.FeedSubscription, 0)
	for rows.Next() {
		subscription := &models.FeedSubscription{}
		err := rows.Scan(
			&subscription.ID,
			&subscription.UserID,
			&subscription.FeedID,
			&subscription.SubscribedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subscriptions: %w", err)
	}

	return subscriptions, nil
}

// GetSubscriberIDs retrieves all user IDs subscribed to a feed
func (r *feedSubscriptionRepository) GetSubscriberIDs(ctx context.Context, feedID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT user_id
		FROM feed_subscriptions
		WHERE feed_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscriber IDs: %w", err)
	}
	defer rows.Close()

	userIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user IDs: %w", err)
	}

	return userIDs, nil
}

// CountByUserID counts the number of subscriptions for a user
func (r *feedSubscriptionRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM feed_subscriptions WHERE user_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count subscriptions: %w", err)
	}

	return count, nil
}

// Delete deletes a subscription
func (r *feedSubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM feed_subscriptions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}

// DeleteByUserAndFeed deletes a subscription by user and feed
func (r *feedSubscriptionRepository) DeleteByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) error {
	query := `DELETE FROM feed_subscriptions WHERE user_id = $1 AND feed_id = $2`

	result, err := r.db.ExecContext(ctx, query, userID, feedID)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subscription not found")
	}

	return nil
}
