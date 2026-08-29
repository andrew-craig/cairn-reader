package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	address := &models.EmailAddress{
		UserID:    userID,
		LocalPart: "abc12345",
	}

	mock.ExpectExec("INSERT INTO email_addresses").
		WithArgs(sqlmock.AnyArg(), userID, "abc12345", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, address)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, address.ID)
	assert.False(t, address.CreatedAt.IsZero())

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddressRepository_GetByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	expectedID := uuid.New()
	expectedTime := time.Now()

	rows := sqlmock.NewRows([]string{"id", "user_id", "local_part", "created_at"}).
		AddRow(expectedID, userID, "abc12345", expectedTime)

	mock.ExpectQuery("SELECT (.+) FROM email_addresses WHERE user_id").
		WithArgs(userID).
		WillReturnRows(rows)

	address, err := repo.GetByUserID(ctx, userID)
	assert.NoError(t, err)
	require.NotNil(t, address)
	assert.Equal(t, expectedID, address.ID)
	assert.Equal(t, userID, address.UserID)
	assert.Equal(t, "abc12345", address.LocalPart)
	assert.Equal(t, expectedTime.Unix(), address.CreatedAt.Unix())

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddressRepository_GetByUserID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery("SELECT (.+) FROM email_addresses WHERE user_id").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "local_part", "created_at"}))

	address, err := repo.GetByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, address)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddressRepository_GetByLocalPart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	expectedID := uuid.New()
	expectedTime := time.Now()
	localPart := "xyz98765"

	rows := sqlmock.NewRows([]string{"id", "user_id", "local_part", "created_at"}).
		AddRow(expectedID, userID, localPart, expectedTime)

	mock.ExpectQuery("SELECT (.+) FROM email_addresses WHERE local_part").
		WithArgs(localPart).
		WillReturnRows(rows)

	address, err := repo.GetByLocalPart(ctx, localPart)
	assert.NoError(t, err)
	require.NotNil(t, address)
	assert.Equal(t, expectedID, address.ID)
	assert.Equal(t, userID, address.UserID)
	assert.Equal(t, localPart, address.LocalPart)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddressRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM email_addresses WHERE user_id").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, userID)
	assert.NoError(t, err)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAddressRepository_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAddressRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectExec("DELETE FROM email_addresses WHERE user_id").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(ctx, userID)
	assert.NoError(t, err) // Should not error when nothing to delete

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
