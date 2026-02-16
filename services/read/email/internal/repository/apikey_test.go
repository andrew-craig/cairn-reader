package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepository_ValidateKey_Valid(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	apiKey := "test-api-key-12345"
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	id := uuid.New().String()
	rows := sqlmock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectQuery("UPDATE api_keys SET last_used_at").
		WithArgs(sqlmock.AnyArg(), keyHash).
		WillReturnRows(rows)

	valid, err := repo.ValidateKey(ctx, apiKey)
	assert.NoError(t, err)
	assert.True(t, valid)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAPIKeyRepository_ValidateKey_Invalid(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	apiKey := "invalid-api-key"
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	mock.ExpectQuery("UPDATE api_keys SET last_used_at").
		WithArgs(sqlmock.AnyArg(), keyHash).
		WillReturnError(sql.ErrNoRows)

	valid, err := repo.ValidateKey(ctx, apiKey)
	assert.NoError(t, err)
	assert.False(t, valid)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAPIKeyRepository_ValidateKey_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	apiKey := "test-api-key"
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	mock.ExpectQuery("UPDATE api_keys SET last_used_at").
		WithArgs(sqlmock.AnyArg(), keyHash).
		WillReturnError(sql.ErrConnDone)

	valid, err := repo.ValidateKey(ctx, apiKey)
	assert.Error(t, err)
	assert.False(t, valid)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAPIKeyRepository_GetKeyHashByName_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	keyName := "cloudflare-worker"
	expectedHash := "abc123hash456"

	rows := sqlmock.NewRows([]string{"key_hash"}).AddRow(expectedHash)

	mock.ExpectQuery("SELECT key_hash FROM api_keys WHERE key_name").
		WithArgs(keyName).
		WillReturnRows(rows)

	keyHash, err := repo.GetKeyHashByName(ctx, keyName)
	assert.NoError(t, err)
	assert.Equal(t, expectedHash, keyHash)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAPIKeyRepository_GetKeyHashByName_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	keyName := "nonexistent-key"

	mock.ExpectQuery("SELECT key_hash FROM api_keys WHERE key_name").
		WithArgs(keyName).
		WillReturnError(sql.ErrNoRows)

	keyHash, err := repo.GetKeyHashByName(ctx, keyName)
	assert.NoError(t, err)
	assert.Empty(t, keyHash)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestAPIKeyRepository_GetKeyHashByName_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	keyName := "test-key"

	mock.ExpectQuery("SELECT key_hash FROM api_keys WHERE key_name").
		WithArgs(keyName).
		WillReturnError(sql.ErrConnDone)

	keyHash, err := repo.GetKeyHashByName(ctx, keyName)
	assert.Error(t, err)
	assert.Empty(t, keyHash)

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
