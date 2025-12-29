package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/andrew-craig/cairn/services/read/content/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserContentRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	userID := uuid.New()
	contentID := uuid.New()
	now := time.Now()

	uc := &models.UserContent{
		ID:             ucID,
		UserID:         userID,
		ContentID:      contentID,
		Status:         models.StatusUnread,
		ScrollPosition: 0,
		IsFavorite:     false,
	}

	mock.ExpectQuery(`INSERT INTO user_contents`).
		WithArgs(
			sqlmock.AnyArg(), // id
			uc.UserID,
			uc.ContentID,
			uc.Status,
			uc.ScrollPosition,
			uc.IsFavorite,
			sqlmock.AnyArg(), // added_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "added_at", "updated_at"}).
			AddRow(ucID, now, now))

	err = repo.Create(ctx, uc)
	assert.NoError(t, err)
	assert.NotNil(t, uc.AddedAt)
	assert.NotNil(t, uc.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	userID := uuid.New()
	contentID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE id = \$1`).
		WithArgs(ucID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).AddRow(
			ucID, userID, contentID, models.StatusUnread, 0, false, now, now,
		))

	result, err := repo.GetByID(ctx, ucID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ucID, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, contentID, result.ContentID)
	assert.Equal(t, models.StatusUnread, result.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE id = \$1`).
		WithArgs(ucID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, ucID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_GetByUserAndContent_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	userID := uuid.New()
	contentID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE user_id = \$1 AND content_id = \$2`).
		WithArgs(userID, contentID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).AddRow(
			ucID, userID, contentID, models.StatusRead, 100, true, now, now,
		))

	result, err := repo.GetByUserAndContent(ctx, userID, contentID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusRead, result.Status)
	assert.Equal(t, 100, result.ScrollPosition)
	assert.True(t, result.IsFavorite)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_GetByUserAndContent_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	contentID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE user_id = \$1 AND content_id = \$2`).
		WithArgs(userID, contentID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByUserAndContent(ctx, userID, contentID)
	assert.NoError(t, err)
	assert.Nil(t, result) // Not found is not an error for this query
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_ListByUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE user_id = \$1 ORDER BY added_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(userID, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).
			AddRow(uuid.New(), userID, uuid.New(), models.StatusUnread, 0, false, now, now).
			AddRow(uuid.New(), userID, uuid.New(), models.StatusRead, 50, true, now, now))

	results, err := repo.ListByUser(ctx, userID, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, models.StatusUnread, results[0].Status)
	assert.Equal(t, models.StatusRead, results[1].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_ListByUserWithFilter_StatusOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	status := models.StatusUnread
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE user_id = \$1 AND status = \$2 ORDER BY added_at DESC LIMIT \$3 OFFSET \$4`).
		WithArgs(userID, status, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).
			AddRow(uuid.New(), userID, uuid.New(), models.StatusUnread, 0, false, now, now))

	results, err := repo.ListByUserWithFilter(ctx, userID, &status, nil, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, models.StatusUnread, results[0].Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_ListByUserWithFilter_FavoriteOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	isFavorite := true
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE user_id = \$1 AND is_favorite = \$2 ORDER BY added_at DESC LIMIT \$3 OFFSET \$4`).
		WithArgs(userID, isFavorite, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).
			AddRow(uuid.New(), userID, uuid.New(), models.StatusRead, 100, true, now, now))

	results, err := repo.ListByUserWithFilter(ctx, userID, nil, &isFavorite, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].IsFavorite)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_ListByUserWithFilter_BothFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	status := models.StatusUnread
	isFavorite := true
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM user_contents WHERE user_id = \$1 AND status = \$2 AND is_favorite = \$3 ORDER BY added_at DESC LIMIT \$4 OFFSET \$5`).
		WithArgs(userID, status, isFavorite, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).
			AddRow(uuid.New(), userID, uuid.New(), models.StatusUnread, 0, true, now, now))

	results, err := repo.ListByUserWithFilter(ctx, userID, &status, &isFavorite, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, models.StatusUnread, results[0].Status)
	assert.True(t, results[0].IsFavorite)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_Update_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	now := time.Now()

	uc := &models.UserContent{
		ID:             ucID,
		Status:         models.StatusRead,
		ScrollPosition: 250,
		IsFavorite:     true,
	}

	mock.ExpectQuery(`UPDATE user_contents SET`).
		WithArgs(
			ucID,
			uc.Status,
			uc.ScrollPosition,
			uc.IsFavorite,
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

	err = repo.Update(ctx, uc)
	assert.NoError(t, err)
	assert.NotNil(t, uc.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_UpdateMetadata_AllFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	status := models.StatusArchived
	scrollPosition := 500
	isFavorite := true

	mock.ExpectExec(`UPDATE user_contents SET updated_at = \$1, status = \$2, scroll_position = \$3, is_favorite = \$4 WHERE id = \$5`).
		WithArgs(sqlmock.AnyArg(), status, scrollPosition, isFavorite, ucID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateMetadata(ctx, ucID, &status, &scrollPosition, &isFavorite)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_UpdateMetadata_StatusOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	status := models.StatusRead

	mock.ExpectExec(`UPDATE user_contents SET updated_at = \$1, status = \$2 WHERE id = \$3`).
		WithArgs(sqlmock.AnyArg(), status, ucID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateMetadata(ctx, ucID, &status, nil, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_UpdateMetadata_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	status := models.StatusRead

	mock.ExpectExec(`UPDATE user_contents SET`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateMetadata(ctx, ucID, &status, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_Delete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	contentID := uuid.New()

	mock.ExpectExec(`DELETE FROM user_contents WHERE user_id = \$1 AND content_id = \$2`).
		WithArgs(userID, contentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, userID, contentID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_Delete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	contentID := uuid.New()

	mock.ExpectExec(`DELETE FROM user_contents WHERE user_id = \$1 AND content_id = \$2`).
		WithArgs(userID, contentID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(ctx, userID, contentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_CountByUser_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_contents WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.CountByUser(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_Search_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	query := "golang"
	now := time.Now()

	mock.ExpectQuery(`SELECT uc\.id, uc\.user_id, uc\.content_id, uc\.status, uc\.scroll_position, uc\.is_favorite, uc\.added_at, uc\.updated_at FROM user_contents uc JOIN contents c`).
		WithArgs(userID, query, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "content_id", "status", "scroll_position", "is_favorite", "added_at", "updated_at",
		}).
			AddRow(uuid.New(), userID, uuid.New(), models.StatusUnread, 0, false, now, now))

	results, err := repo.Search(ctx, userID, query, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_BulkCreate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	userContents := []*models.UserContent{
		{
			UserID:    userID,
			ContentID: uuid.New(),
			Status:    models.StatusUnread,
		},
		{
			UserID:    userID,
			ContentID: uuid.New(),
			Status:    models.StatusUnread,
		},
	}

	mock.ExpectBegin()
	stmt := mock.ExpectPrepare(`INSERT INTO user_contents`)
	for range userContents {
		stmt.ExpectQuery().
			WillReturnRows(sqlmock.NewRows([]string{"id", "added_at", "updated_at"}).
				AddRow(uuid.New(), now, now))
	}
	mock.ExpectCommit()

	err = repo.BulkCreate(ctx, userContents)
	assert.NoError(t, err)
	for _, uc := range userContents {
		assert.NotEqual(t, uuid.Nil, uc.ID)
		assert.NotNil(t, uc.AddedAt)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_BulkCreate_ConflictHandling(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()

	userContents := []*models.UserContent{
		{
			UserID:    userID,
			ContentID: uuid.New(),
			Status:    models.StatusUnread,
		},
	}

	mock.ExpectBegin()
	stmt := mock.ExpectPrepare(`INSERT INTO user_contents`)
	// ON CONFLICT DO NOTHING returns no rows
	stmt.ExpectQuery().
		WillReturnRows(sqlmock.NewRows([]string{"id", "added_at", "updated_at"}).
			AddRow(uuid.New(), now, now))
	mock.ExpectCommit()

	err = repo.BulkCreate(ctx, userContents)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_CreateWithTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	userID := uuid.New()
	contentID := uuid.New()
	now := time.Now()

	uc := &models.UserContent{
		ID:        ucID,
		UserID:    userID,
		ContentID: contentID,
		Status:    models.StatusUnread,
	}

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)

	mock.ExpectQuery(`INSERT INTO user_contents`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "added_at", "updated_at"}).
			AddRow(ucID, now, now))

	err = repo.CreateWithTx(ctx, tx, uc)
	assert.NoError(t, err)
	assert.NotNil(t, uc.AddedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_UpdateWithTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	ucID := uuid.New()
	now := time.Now()

	uc := &models.UserContent{
		ID:             ucID,
		Status:         models.StatusRead,
		ScrollPosition: 100,
		IsFavorite:     true,
	}

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)

	mock.ExpectQuery(`UPDATE user_contents SET`).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

	err = repo.UpdateWithTx(ctx, tx, uc)
	assert.NoError(t, err)
	assert.NotNil(t, uc.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserContentRepository_DeleteWithTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserContentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	contentID := uuid.New()

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)

	mock.ExpectExec(`DELETE FROM user_contents WHERE user_id = \$1 AND content_id = \$2`).
		WithArgs(userID, contentID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.DeleteWithTx(ctx, tx, userID, contentID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
