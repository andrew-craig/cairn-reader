package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	rawEmailID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	payload := map[string]interface{}{
		"title": "Test Email",
		"html":  "<p>Test content</p>",
	}

	outbox := &models.ContentOutbox{
		RawEmailID:     rawEmailID,
		ContentPayload: payload,
		UserID:         userID,
		DeliveryStatus: models.DeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     6,
		NextRetryAt:    now,
	}

	payloadJSON, _ := json.Marshal(payload)

	mock.ExpectExec("INSERT INTO content_outbox").
		WithArgs(
			sqlmock.AnyArg(), // id
			rawEmailID,
			payloadJSON,
			userID,
			models.DeliveryStatusPending,
			0, // retry_count
			6, // max_retries
			now,
			nil,              // last_error
			nil,              // content_service_id
			sqlmock.AnyArg(), // created_at
			nil,              // delivered_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, outbox)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, outbox.ID)
	assert.False(t, outbox.CreatedAt.IsZero())

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestOutboxRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	id := uuid.New()
	rawEmailID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	payload := map[string]interface{}{
		"title": "Test Email",
	}
	payloadJSON, _ := json.Marshal(payload)

	rows := sqlmock.NewRows([]string{
		"id", "raw_email_id", "content_payload", "user_id",
		"delivery_status", "retry_count", "max_retries", "next_retry_at", "last_error",
		"content_service_id", "created_at", "delivered_at",
	}).AddRow(
		id, rawEmailID, payloadJSON, userID,
		models.DeliveryStatusPending, 0, 6, now, nil,
		nil, now, nil,
	)

	mock.ExpectQuery("SELECT .+ FROM content_outbox WHERE id").
		WithArgs(id).
		WillReturnRows(rows)

	outbox, err := repo.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, outbox.ID)
	assert.Equal(t, rawEmailID, outbox.RawEmailID)
	assert.Equal(t, userID, outbox.UserID)
	assert.Equal(t, models.DeliveryStatusPending, outbox.DeliveryStatus)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestOutboxRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM content_outbox WHERE id").
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	outbox, err := repo.GetByID(ctx, id)
	assert.NoError(t, err) // GetByID returns nil, nil when not found
	assert.Nil(t, outbox)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestOutboxRepository_GetPendingEntries(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	id1 := uuid.New()
	id2 := uuid.New()
	rawEmailID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	payload := map[string]interface{}{"title": "Test"}
	payloadJSON, _ := json.Marshal(payload)

	rows := sqlmock.NewRows([]string{
		"id", "raw_email_id", "content_payload", "user_id",
		"delivery_status", "retry_count", "max_retries", "next_retry_at", "last_error",
		"content_service_id", "created_at", "delivered_at",
	}).
		AddRow(id1, rawEmailID, payloadJSON, userID,
			models.DeliveryStatusPending, 0, 6, now, nil,
			nil, now, nil).
		AddRow(id2, rawEmailID, payloadJSON, userID,
			models.DeliveryStatusPending, 0, 6, now, nil,
			nil, now, nil)

	mock.ExpectQuery("SELECT .+ FROM content_outbox WHERE delivery_status").
		WithArgs(models.DeliveryStatusPending, models.DeliveryStatusSending, sqlmock.AnyArg(), 10).
		WillReturnRows(rows)

	entries, err := repo.GetPendingEntries(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, id1, entries[0].ID)
	assert.Equal(t, id2, entries[1].ID)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestOutboxRepository_UpdateDeliveryStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	id := uuid.New()
	contentServiceID := uuid.New()
	now := time.Now()

	mock.ExpectExec("UPDATE content_outbox SET delivery_status").
		WithArgs(models.DeliveryStatusDelivered, &contentServiceID, &now, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateDeliveryStatus(ctx, id, models.DeliveryStatusDelivered, &contentServiceID, &now)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestOutboxRepository_UpdateRetryInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	id := uuid.New()
	nextRetryAt := time.Now().Add(5 * time.Minute)
	errorMsg := "Connection timeout"

	mock.ExpectExec("UPDATE content_outbox SET retry_count").
		WithArgs(3, nextRetryAt, errorMsg, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateRetryInfo(ctx, id, 3, nextRetryAt, errorMsg)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestOutboxRepository_DeleteDelivered(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutboxRepository(db)
	ctx := context.Background()

	duration := 7 * 24 * time.Hour

	mock.ExpectExec("DELETE FROM content_outbox WHERE delivery_status").
		WithArgs(models.DeliveryStatusDelivered, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 10))

	count, err := repo.DeleteDelivered(ctx, duration)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), count)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
