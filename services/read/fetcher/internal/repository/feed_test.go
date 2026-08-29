package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	now := time.Now()
	title := "Test Feed"
	description := "Test Description"

	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     "https://example.com/feed",
		Title:       &title,
		Description: &description,
		PollingTier: models.PollingTierActive,
		NextPollAt:  now,
		Status:      models.FeedStatusActive,
	}

	mock.ExpectExec(`INSERT INTO feeds`).
		WithArgs(
			feedID,
			feed.FeedURL,
			feed.Title,
			feed.Description,
			feed.SiteURL,
			feed.PollingTier,
			feed.LastFetchedAt,
			feed.LastPublishedAt,
			sqlmock.AnyArg(), // next_poll_at
			feed.Status,
			feed.ConsecutiveErrorDays,
			feed.LastErrorAt,
			feed.LastErrorMessage,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, feed)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	now := time.Now()
	title := "Test Feed"

	mock.ExpectQuery(`SELECT (.+) FROM feeds WHERE id = \$1`).
		WithArgs(feedID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_url", "title", "description", "site_url",
			"polling_tier", "last_fetched_at", "last_published_at", "next_poll_at",
			"status", "consecutive_error_days", "last_error_at", "last_error_message",
			"created_at", "updated_at",
		}).AddRow(
			feedID, "https://example.com/feed", title, nil, nil,
			models.PollingTierActive, nil, nil, now,
			models.FeedStatusActive, 0, nil, nil,
			now, now,
		))

	result, err := repo.GetByID(ctx, feedID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, feedID, result.ID)
	assert.Equal(t, "https://example.com/feed", result.FeedURL)
	assert.Equal(t, &title, result.Title)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM feeds WHERE id = \$1`).
		WithArgs(feedID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, feedID)
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_GetByURL_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	feedURL := "https://example.com/feed"
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feeds WHERE feed_url = \$1`).
		WithArgs(feedURL).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_url", "title", "description", "site_url",
			"polling_tier", "last_fetched_at", "last_published_at", "next_poll_at",
			"status", "consecutive_error_days", "last_error_at", "last_error_message",
			"created_at", "updated_at",
		}).AddRow(
			feedID, feedURL, nil, nil, nil,
			models.PollingTierActive, nil, nil, now,
			models.FeedStatusActive, 0, nil, nil,
			now, now,
		))

	result, err := repo.GetByURL(ctx, feedURL)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, feedURL, result.FeedURL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_Update_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	now := time.Now()
	title := "Updated Title"

	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     "https://example.com/feed",
		Title:       &title,
		PollingTier: models.PollingTierModerate,
		NextPollAt:  now,
		Status:      models.FeedStatusActive,
	}

	mock.ExpectExec(`UPDATE feeds SET`).
		WithArgs(
			feedID,
			feed.FeedURL,
			feed.Title,
			feed.Description,
			feed.SiteURL,
			feed.PollingTier,
			feed.LastFetchedAt,
			feed.LastPublishedAt,
			feed.NextPollAt,
			feed.Status,
			feed.ConsecutiveErrorDays,
			feed.LastErrorAt,
			feed.LastErrorMessage,
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(ctx, feed)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_Update_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feed := &models.Feed{
		ID:          uuid.New(),
		FeedURL:     "https://example.com/feed",
		PollingTier: models.PollingTierActive,
		NextPollAt:  time.Now(),
		Status:      models.FeedStatusActive,
	}

	mock.ExpectExec(`UPDATE feeds SET`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Update(ctx, feed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_UpdatePollingInfo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	now := time.Now()
	lastFetched := now.Add(-1 * time.Hour)
	lastPublished := now.Add(-2 * time.Hour)

	mock.ExpectExec(`UPDATE feeds SET last_fetched_at = \$2, last_published_at = \$3, next_poll_at = \$4, polling_tier = \$5`).
		WithArgs(feedID, lastFetched, lastPublished, now, models.PollingTierActive, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdatePollingInfo(ctx, feedID, &lastFetched, &lastPublished, now, models.PollingTierActive)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_UpdateErrorInfo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()
	now := time.Now()
	errorMsg := "Connection timeout"

	mock.ExpectExec(`UPDATE feeds SET consecutive_error_days = \$2, last_error_at = \$3, last_error_message = \$4`).
		WithArgs(feedID, 3, now, errorMsg, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateErrorInfo(ctx, feedID, 3, &now, &errorMsg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_UpdateStatus_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()

	mock.ExpectExec(`UPDATE feeds SET status = \$2, updated_at = \$3 WHERE id = \$1`).
		WithArgs(feedID, models.FeedStatusDisabled, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateStatus(ctx, feedID, models.FeedStatusDisabled)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_GetFeedsDueForPolling_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	now := time.Now()

	mock.ExpectQuery(`FOR UPDATE SKIP LOCKED.*UPDATE feeds`).
		WithArgs(sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_url", "title", "description", "site_url",
			"polling_tier", "last_fetched_at", "last_published_at", "next_poll_at",
			"status", "consecutive_error_days", "last_error_at", "last_error_message",
			"created_at", "updated_at",
		}).
			AddRow(uuid.New(), "https://example.com/feed1", nil, nil, nil,
				models.PollingTierActive, nil, nil, now,
				models.FeedStatusActive, 0, nil, nil, now, now).
			AddRow(uuid.New(), "https://example.com/feed2", nil, nil, nil,
				models.PollingTierModerate, nil, nil, now,
				models.FeedStatusActive, 0, nil, nil, now, now))

	results, err := repo.GetFeedsDueForPolling(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_GetFeedsForTierUpdate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feeds WHERE status = 'active' ORDER BY last_published_at DESC NULLS LAST`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_url", "title", "description", "site_url",
			"polling_tier", "last_fetched_at", "last_published_at", "next_poll_at",
			"status", "consecutive_error_days", "last_error_at", "last_error_message",
			"created_at", "updated_at",
		}).
			AddRow(uuid.New(), "https://example.com/feed1", nil, nil, nil,
				models.PollingTierActive, nil, &now, now,
				models.FeedStatusActive, 0, nil, nil, now, now).
			AddRow(uuid.New(), "https://example.com/feed2", nil, nil, nil,
				models.PollingTierQuiet, nil, nil, now,
				models.FeedStatusActive, 0, nil, nil, now, now))

	results, err := repo.GetFeedsForTierUpdate(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()

	mock.ExpectExec(`DELETE FROM feeds WHERE id = \$1`).
		WithArgs(feedID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, feedID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedRepository_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedRepository(db)
	ctx := context.Background()

	feedID := uuid.New()

	mock.ExpectExec(`DELETE FROM feeds WHERE id = \$1`).
		WithArgs(feedID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(ctx, feedID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
