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

func TestSenderRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	senderName := "Test Sender"
	sender := &models.EmailSender{
		UserID:      userID,
		SenderEmail: "sender@example.com",
		SenderName:  &senderName,
		EmailCount:  0,
	}

	mock.ExpectExec("INSERT INTO email_senders").
		WithArgs(
			sqlmock.AnyArg(), // id
			userID,
			"sender@example.com",
			&senderName,
			0,                // email_count
			nil,              // last_received_at
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, sender)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, sender.ID)
	assert.False(t, sender.CreatedAt.IsZero())
	assert.False(t, sender.UpdatedAt.IsZero())

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestSenderRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	id := uuid.New()
	userID := uuid.New()
	senderName := "Test Sender"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "sender_email", "sender_name",
		"email_count", "last_received_at", "created_at", "updated_at",
	}).AddRow(
		id, userID, "sender@example.com", senderName,
		5, now, now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM email_senders WHERE id").
		WithArgs(id).
		WillReturnRows(rows)

	sender, err := repo.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, sender.ID)
	assert.Equal(t, userID, sender.UserID)
	assert.Equal(t, "sender@example.com", sender.SenderEmail)
	assert.Equal(t, senderName, *sender.SenderName)
	assert.Equal(t, 5, sender.EmailCount)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestSenderRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	id := uuid.New()

	mock.ExpectQuery("SELECT .+ FROM email_senders WHERE id").
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	sender, err := repo.GetByID(ctx, id)
	assert.NoError(t, err) // GetByID returns nil, nil when not found
	assert.Nil(t, sender)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestSenderRepository_GetByUserAndEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	id := uuid.New()
	userID := uuid.New()
	senderEmail := "sender@example.com"
	senderName := "Test Sender"
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "sender_email", "sender_name",
		"email_count", "last_received_at", "created_at", "updated_at",
	}).AddRow(
		id, userID, senderEmail, senderName,
		3, now, now, now,
	)

	mock.ExpectQuery("SELECT .+ FROM email_senders WHERE user_id").
		WithArgs(userID, senderEmail).
		WillReturnRows(rows)

	sender, err := repo.GetByUserAndEmail(ctx, userID, senderEmail)
	assert.NoError(t, err)
	assert.Equal(t, id, sender.ID)
	assert.Equal(t, userID, sender.UserID)
	assert.Equal(t, senderEmail, sender.SenderEmail)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestSenderRepository_Upsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	returnedID := uuid.New()
	senderName := "Test Sender"
	now := time.Now()
	sender := &models.EmailSender{
		UserID:         userID,
		SenderEmail:    "sender@example.com",
		SenderName:     &senderName,
		EmailCount:     1,
		LastReceivedAt: &now,
	}

	// Upsert uses QueryRowContext with RETURNING clause
	rows := sqlmock.NewRows([]string{"id", "email_count", "created_at", "updated_at"}).
		AddRow(returnedID, 2, now, now)

	mock.ExpectQuery("INSERT INTO email_senders .+ ON CONFLICT").
		WithArgs(
			sqlmock.AnyArg(), // id
			userID,
			"sender@example.com",
			&senderName,
			1,
			&now,
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnRows(rows)

	err = repo.Upsert(ctx, sender)
	assert.NoError(t, err)
	assert.Equal(t, returnedID, sender.ID)
	assert.Equal(t, 2, sender.EmailCount) // Incremented by DB

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestSenderRepository_ListByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "user_id", "sender_email", "sender_name",
		"email_count", "last_received_at", "created_at", "updated_at",
	}).
		AddRow(uuid.New(), userID, "sender1@example.com", "Sender 1", 10, now, now, now).
		AddRow(uuid.New(), userID, "sender2@example.com", "Sender 2", 5, now, now, now)

	mock.ExpectQuery("SELECT .+ FROM email_senders WHERE user_id").
		WithArgs(userID, 10, 0).
		WillReturnRows(rows)

	senders, err := repo.ListByUser(ctx, userID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, senders, 2)
	assert.Equal(t, "sender1@example.com", senders[0].SenderEmail)
	assert.Equal(t, "sender2@example.com", senders[1].SenderEmail)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestSenderRepository_IncrementEmailCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewSenderRepository(db)
	ctx := context.Background()

	id := uuid.New()
	now := time.Now()

	mock.ExpectExec("UPDATE email_senders SET").
		WithArgs(now, sqlmock.AnyArg(), id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.IncrementEmailCount(ctx, id, now)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
