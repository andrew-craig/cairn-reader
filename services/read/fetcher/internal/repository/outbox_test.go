package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	outboxID := uuid.New()
	feedItemID := uuid.New()
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}
	payload := map[string]interface{}{
		"title": "Test Article",
		"url":   "https://example.com",
	}

	outbox := &models.ContentOutbox{
		ID:             outboxID,
		FeedItemID:     feedItemID,
		ContentPayload: payload,
		UserIDs:        userIDs,
		DeliveryStatus: models.DeliveryStatusPending,
		MaxRetries:     6,
		NextRetryAt:    time.Now(),
	}

	mock.ExpectExec(`INSERT INTO content_outbox`).
		WithArgs(
			outboxID,
			feedItemID,
			sqlmock.AnyArg(), // content_payload JSON
			sqlmock.AnyArg(), // user_ids array
			outbox.DeliveryStatus,
			outbox.RetryCount,
			outbox.MaxRetries,
			sqlmock.AnyArg(), // next_retry_at
			outbox.LastError,
			outbox.ContentServiceID,
			sqlmock.AnyArg(), // created_at
			outbox.DeliveredAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, outbox)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	outboxID := uuid.New()
	feedItemID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	now := time.Now()

	payload := map[string]interface{}{
		"title": "Test",
	}
	payloadJSON, _ := json.Marshal(payload)

	userIDStrings := []string{userID1.String(), userID2.String()}

	mock.ExpectQuery(`SELECT (.+) FROM content_outbox WHERE id = \$1`).
		WithArgs(outboxID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_item_id", "content_payload", "user_ids",
			"delivery_status", "retry_count", "max_retries", "next_retry_at", "last_error",
			"content_service_id", "created_at", "delivered_at",
		}).AddRow(
			outboxID, feedItemID, payloadJSON, pq.Array(userIDStrings),
			models.DeliveryStatusPending, 0, 6, now, nil,
			nil, now, nil,
		))

	result, err := repo.GetByID(ctx, outboxID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, outboxID, result.ID)
	assert.Equal(t, feedItemID, result.FeedItemID)
	assert.Len(t, result.UserIDs, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	outboxID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM content_outbox WHERE id = \$1`).
		WithArgs(outboxID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, outboxID)
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_GetPendingEntries_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	now := time.Now()
	payload := map[string]interface{}{"title": "Test"}
	payloadJSON, _ := json.Marshal(payload)
	userIDStrings := []string{uuid.New().String()}

	mock.ExpectQuery(`SELECT (.+) FROM content_outbox WHERE delivery_status = 'pending' AND next_retry_at <= \$1 ORDER BY next_retry_at ASC LIMIT \$2`).
		WithArgs(sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_item_id", "content_payload", "user_ids",
			"delivery_status", "retry_count", "max_retries", "next_retry_at", "last_error",
			"content_service_id", "created_at", "delivered_at",
		}).AddRow(
			uuid.New(), uuid.New(), payloadJSON, pq.Array(userIDStrings),
			models.DeliveryStatusPending, 0, 6, now, nil,
			nil, now, nil,
		))

	results, err := repo.GetPendingEntries(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, models.DeliveryStatusPending, results[0].DeliveryStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_UpdateDeliveryStatus_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	outboxID := uuid.New()
	contentServiceID := uuid.New()
	now := time.Now()
	errorMsg := "Connection failed"

	mock.ExpectExec(`UPDATE content_outbox SET delivery_status = \$2, content_service_id = \$3, delivered_at = \$4, last_error = \$5 WHERE id = \$1`).
		WithArgs(outboxID, models.DeliveryStatusFailed, contentServiceID, now, errorMsg).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateDeliveryStatus(ctx, outboxID, models.DeliveryStatusFailed, &contentServiceID, &now, &errorMsg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_IncrementRetryCount_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	outboxID := uuid.New()
	nextRetry := time.Now().Add(5 * time.Minute)
	errorMsg := "Temporary failure"

	mock.ExpectExec(`UPDATE content_outbox SET retry_count = retry_count \+ 1, next_retry_at = \$2, last_error = \$3`).
		WithArgs(outboxID, nextRetry, errorMsg).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.IncrementRetryCount(ctx, outboxID, nextRetry, errorMsg)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_DeleteOldDeliveredEntries_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	olderThan := time.Now().Add(-7 * 24 * time.Hour)

	mock.ExpectExec(`DELETE FROM content_outbox WHERE id IN`).
		WithArgs(olderThan, 100).
		WillReturnResult(sqlmock.NewResult(0, 25))

	count, err := repo.DeleteOldDeliveredEntries(ctx, olderThan, 100)
	assert.NoError(t, err)
	assert.Equal(t, 25, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_GetFailedEntries_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	now := time.Now()
	payload := map[string]interface{}{"title": "Failed"}
	payloadJSON, _ := json.Marshal(payload)
	userIDStrings := []string{uuid.New().String()}

	mock.ExpectQuery(`SELECT (.+) FROM content_outbox WHERE delivery_status = 'failed' ORDER BY created_at DESC LIMIT \$1`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "feed_item_id", "content_payload", "user_ids",
			"delivery_status", "retry_count", "max_retries", "next_retry_at", "last_error",
			"content_service_id", "created_at", "delivered_at",
		}).AddRow(
			uuid.New(), uuid.New(), payloadJSON, pq.Array(userIDStrings),
			models.DeliveryStatusFailed, 6, 6, now, "Max retries exceeded",
			nil, now, nil,
		))

	results, err := repo.GetFailedEntries(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, models.DeliveryStatusFailed, results[0].DeliveryStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	outboxID := uuid.New()

	mock.ExpectExec(`DELETE FROM content_outbox WHERE id = \$1`).
		WithArgs(outboxID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, outboxID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepository_GetMetrics_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(`SELECT (.+) FROM content_outbox`).
		WillReturnRows(sqlmock.NewRows([]string{
			"pending_count", "sending_count", "delivered_count", "failed_count", "total_count",
		}).AddRow(5, 2, 100, 3, 110))

	metrics, err := repo.GetMetrics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), metrics.PendingCount)
	assert.Equal(t, int64(2), metrics.SendingCount)
	assert.Equal(t, int64(100), metrics.DeliveredCount)
	assert.Equal(t, int64(3), metrics.FailedCount)
	assert.Equal(t, int64(110), metrics.TotalCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}
