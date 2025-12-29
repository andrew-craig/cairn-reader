package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedSubscriptionRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	subID := uuid.New()
	userID := uuid.New()
	feedID := uuid.New()

	sub := &models.FeedSubscription{
		ID:     subID,
		UserID: userID,
		FeedID: feedID,
	}

	mock.ExpectExec(`INSERT INTO feed_subscriptions`).
		WithArgs(subID, userID, feedID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	subID := uuid.New()
	userID := uuid.New()
	feedID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_subscriptions WHERE id = \$1`).
		WithArgs(subID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "feed_id", "subscribed_at",
		}).AddRow(subID, userID, feedID, now))

	result, err := repo.GetByID(ctx, subID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, subID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, feedID, result.FeedID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	subID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM feed_subscriptions WHERE id = \$1`).
		WithArgs(subID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, subID)
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_GetByUserAndFeed_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	subID := uuid.New()
	userID := uuid.New()
	feedID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_subscriptions WHERE user_id = \$1 AND feed_id = \$2`).
		WithArgs(userID, feedID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "feed_id", "subscribed_at",
		}).AddRow(subID, userID, feedID, now))

	result, err := repo.GetByUserAndFeed(ctx, userID, feedID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, feedID, result.FeedID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_GetByUserID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_subscriptions WHERE user_id = \$1 ORDER BY subscribed_at DESC`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "feed_id", "subscribed_at",
		}).
			AddRow(uuid.New(), userID, uuid.New(), now).
			AddRow(uuid.New(), userID, uuid.New(), now))

	results, err := repo.GetByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_GetByFeedID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_subscriptions WHERE feed_id = \$1 ORDER BY subscribed_at DESC`).
		WithArgs(feedID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "feed_id", "subscribed_at",
		}).
			AddRow(uuid.New(), uuid.New(), feedID, now).
			AddRow(uuid.New(), uuid.New(), feedID, now).
			AddRow(uuid.New(), uuid.New(), feedID, now))

	results, err := repo.GetByFeedID(ctx, feedID)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	for _, sub := range results {
		assert.Equal(t, feedID, sub.FeedID)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_GetSubscriberIDs_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	mock.ExpectQuery(`SELECT user_id FROM feed_subscriptions WHERE feed_id = \$1`).
		WithArgs(feedID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
			AddRow(userID1).
			AddRow(userID2))

	results, err := repo.GetSubscriberIDs(ctx, feedID)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, results, userID1)
	assert.Contains(t, results, userID2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_CountByUserID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM feed_subscriptions WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.CountByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 42, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	subID := uuid.New()

	mock.ExpectExec(`DELETE FROM feed_subscriptions WHERE id = \$1`).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, subID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	subID := uuid.New()

	mock.ExpectExec(`DELETE FROM feed_subscriptions WHERE id = \$1`).
		WithArgs(subID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(ctx, subID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_DeleteByUserAndFeed_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	feedID := uuid.New()

	mock.ExpectExec(`DELETE FROM feed_subscriptions WHERE user_id = \$1 AND feed_id = \$2`).
		WithArgs(userID, feedID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.DeleteByUserAndFeed(ctx, userID, feedID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedSubscriptionRepository_DeleteByUserAndFeed_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedSubscriptionRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	feedID := uuid.New()

	mock.ExpectExec(`DELETE FROM feed_subscriptions WHERE user_id = \$1 AND feed_id = \$2`).
		WithArgs(userID, feedID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.DeleteByUserAndFeed(ctx, userID, feedID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
