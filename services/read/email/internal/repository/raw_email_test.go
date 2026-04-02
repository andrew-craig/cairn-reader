package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawEmailRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	htmlBody := "<html>Test email</html>"
	subject := "Test Subject"
	now := time.Now()

	email := &models.RawEmail{
		UserID:           userID,
		Recipient:        "test@example.com",
		SenderEmail:      "sender@example.com",
		Subject:          &subject,
		HTMLBody:         &htmlBody,
		ReceivedAt:       now,
		ProcessingStatus: models.ProcessingStatusPending,
		RetryCount:       0,
	}

	mock.ExpectExec("INSERT INTO raw_emails").
		WithArgs(
			sqlmock.AnyArg(), // id
			userID,
			nil, // sender_id
			"test@example.com",
			"sender@example.com",
			nil, // sender_name
			&subject,
			&htmlBody,
			nil, // text_body
			now,
			models.ProcessingStatusPending,
			nil,              // content_hash
			0,                // retry_count
			nil,              // last_error
			sqlmock.AnyArg(), // created_at
			nil,              // processed_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, email)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, email.ID)
	assert.False(t, email.CreatedAt.IsZero())

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRawEmailRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	id := uuid.New()
	userID := uuid.New()
	htmlBody := "<html>Test</html>"
	subject := "Test"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "sender_id", "recipient", "sender_email", "sender_name",
		"subject", "html_body", "text_body", "received_at",
		"processing_status", "content_hash", "retry_count", "last_error",
		"created_at", "processed_at",
	}).AddRow(
		id, userID, nil, "test@example.com", "sender@example.com", nil,
		subject, htmlBody, nil, now,
		models.ProcessingStatusPending, nil, 0, nil,
		now, nil,
	)

	mock.ExpectQuery("SELECT .+ FROM raw_emails WHERE id").
		WithArgs(id).
		WillReturnRows(rows)

	email, err := repo.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, email.ID)
	assert.Equal(t, userID, email.UserID)
	assert.Equal(t, "test@example.com", email.Recipient)
	assert.Equal(t, models.ProcessingStatusPending, email.ProcessingStatus)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRawEmailRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM raw_emails WHERE id").
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	email, err := repo.GetByID(ctx, id)
	assert.NoError(t, err) // GetByID returns nil, nil when not found
	assert.Nil(t, email)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRawEmailRepository_GetPendingEmails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	id1 := uuid.New()
	id2 := uuid.New()
	userID := uuid.New()
	htmlBody := "<html>Test</html>"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "sender_id", "recipient", "sender_email", "sender_name",
		"subject", "html_body", "text_body", "received_at",
		"processing_status", "content_hash", "retry_count", "last_error",
		"created_at", "processed_at",
	}).
		AddRow(id1, userID, nil, "test1@example.com", "sender@example.com", nil,
			"Subject 1", htmlBody, nil, now,
			models.ProcessingStatusPending, nil, 0, nil,
			now, nil).
		AddRow(id2, userID, nil, "test2@example.com", "sender@example.com", nil,
			"Subject 2", htmlBody, nil, now,
			models.ProcessingStatusPending, nil, 0, nil,
			now, nil)

	mock.ExpectQuery("SELECT .+ FROM raw_emails WHERE processing_status").
		WithArgs(models.ProcessingStatusPending, 10).
		WillReturnRows(rows)

	emails, err := repo.GetPendingEmails(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, emails, 2)
	assert.Equal(t, id1, emails[0].ID)
	assert.Equal(t, id2, emails[1].ID)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRawEmailRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()

	mock.ExpectExec("UPDATE raw_emails SET processing_status").
		WithArgs(models.ProcessingStatusCompleted, &now, id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateStatus(ctx, id, models.ProcessingStatusCompleted, &now)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRawEmailRepository_UpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	id := uuid.New()
	errorMsg := "Failed to process"

	mock.ExpectExec("UPDATE raw_emails SET retry_count").
		WithArgs(3, errorMsg, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateError(ctx, id, 3, errorMsg)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestRawEmailRepository_DeleteProcessed(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewRawEmailRepository(db)
	ctx := context.Background()

	duration := 7 * 24 * time.Hour

	mock.ExpectExec("DELETE FROM raw_emails WHERE processing_status").
		WithArgs(models.ProcessingStatusCompleted, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))

	count, err := repo.DeleteProcessed(ctx, duration)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
