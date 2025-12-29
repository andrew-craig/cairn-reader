package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedItemRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	itemID := uuid.New()
	feedID := uuid.New()
	guid := "item-guid-123"
	title := "Test Item"

	item := &models.FeedItem{
		ID:               itemID,
		FeedID:           feedID,
		ItemURL:          "https://example.com/article",
		ItemGUID:         &guid,
		ProcessingStatus: models.ProcessingStatusPending,
		Title:            &title,
	}

	mock.ExpectExec(`INSERT INTO feed_items`).
		WithArgs(sqlmock.AnyArg(), feedID, item.ItemURL, item.ItemGUID,
			item.ProcessingStatus, item.ContentHash, item.ContentServiceID,
			item.Title, item.Author, item.PublishedAt, item.Description,
			item.HTTPLastModified, item.HTTPETag, item.LastCheckedAt,
			item.RetryCount, item.LastError, sqlmock.AnyArg(), item.ProcessedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, item)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	itemID := uuid.New()
	feedID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_items WHERE id = \$1`).
		WithArgs(itemID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_id", "item_url", "item_guid",
			"processing_status", "content_hash", "content_service_id",
			"title", "author", "published_at", "description",
			"http_last_modified", "http_etag", "last_checked_at",
			"retry_count", "last_error", "discovered_at", "processed_at",
		}).AddRow(
			itemID, feedID, "https://example.com/article", "guid",
			models.ProcessingStatusPending, nil, nil,
			nil, nil, nil, nil,
			nil, nil, nil,
			0, nil, now, nil,
		))

	result, err := repo.GetByID(ctx, itemID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, itemID, result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_GetByFeedAndGUID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	itemID := uuid.New()
	feedID := uuid.New()
	guid := "test-guid"
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_items WHERE feed_id = \$1 AND item_guid = \$2`).
		WithArgs(feedID, guid).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_id", "item_url", "item_guid",
			"processing_status", "content_hash", "content_service_id",
			"title", "author", "published_at", "description",
			"http_last_modified", "http_etag", "last_checked_at",
			"retry_count", "last_error", "discovered_at", "processed_at",
		}).AddRow(
			itemID, feedID, "https://example.com", guid,
			models.ProcessingStatusCompleted, nil, nil,
			nil, nil, nil, nil,
			nil, nil, nil,
			0, nil, now, &now,
		))

	result, err := repo.GetByFeedAndGUID(ctx, feedID, guid)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, feedID, result.FeedID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_Update_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	item := &models.FeedItem{
		ID:               uuid.New(),
		FeedID:           uuid.New(),
		ItemURL:          "https://example.com",
		ProcessingStatus: models.ProcessingStatusCompleted,
	}

	mock.ExpectExec(`UPDATE feed_items SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(ctx, item)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_UpdateProcessingStatus_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	itemID := uuid.New()
	contentHash := "hash123"
	contentServiceID := uuid.New()
	now := time.Now()

	mock.ExpectExec(`UPDATE feed_items SET processing_status = \$2, content_hash = \$3, content_service_id = \$4, processed_at = \$5 WHERE id = \$1`).
		WithArgs(itemID, models.ProcessingStatusCompleted, contentHash, contentServiceID, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateProcessingStatus(ctx, itemID, models.ProcessingStatusCompleted, &contentHash, &contentServiceID, &now)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_GetPendingItems_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM feed_items WHERE processing_status = 'pending' ORDER BY discovered_at ASC LIMIT \$1`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_id", "item_url", "item_guid",
			"processing_status", "content_hash", "content_service_id",
			"title", "author", "published_at", "description",
			"http_last_modified", "http_etag", "last_checked_at",
			"retry_count", "last_error", "discovered_at", "processed_at",
		}).AddRow(
			uuid.New(), uuid.New(), "https://example.com", "guid",
			models.ProcessingStatusPending, nil, nil,
			nil, nil, nil, nil,
			nil, nil, nil,
			0, nil, now, nil,
		))

	results, err := repo.GetPendingItems(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_DeleteOldCompletedItems_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	olderThan := time.Now().Add(-7 * 24 * time.Hour)

	mock.ExpectExec(`DELETE FROM feed_items WHERE id IN`).
		WithArgs(olderThan, 100).
		WillReturnResult(sqlmock.NewResult(0, 15))

	count, err := repo.DeleteOldCompletedItems(ctx, olderThan, 100)
	assert.NoError(t, err)
	assert.Equal(t, 15, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_DeleteOldFailedItems_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	olderThan := time.Now().Add(-30 * 24 * time.Hour)

	mock.ExpectExec(`DELETE FROM feed_items WHERE id IN`).
		WithArgs(olderThan, 100).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteOldFailedItems(ctx, olderThan, 100)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFeedItemRepository_GetMetrics_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewFeedItemRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT (.+) FROM feed_items`).
		WillReturnRows(sqlmock.NewRows([]string{
			"pending_count", "processing_count", "completed_count", "failed_count", "total_count",
		}).AddRow(10, 5, 100, 3, 118))

	metrics, err := repo.GetMetrics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), metrics.PendingCount)
	assert.Equal(t, int64(5), metrics.ProcessingCount)
	assert.Equal(t, int64(100), metrics.CompletedCount)
	assert.Equal(t, int64(3), metrics.FailedCount)
	assert.Equal(t, int64(118), metrics.TotalCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}
