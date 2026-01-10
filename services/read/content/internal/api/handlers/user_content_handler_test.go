package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserContentRepository is a mock implementation of repository.UserContentRepository
type MockUserContentRepository struct {
	mock.Mock
}

func (m *MockUserContentRepository) Create(ctx context.Context, uc *models.UserContent) error {
	args := m.Called(ctx, uc)
	return args.Error(0)
}

func (m *MockUserContentRepository) BulkCreate(ctx context.Context, userContents []*models.UserContent) error {
	args := m.Called(ctx, userContents)
	return args.Error(0)
}

func (m *MockUserContentRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, userContent *models.UserContent) error {
	args := m.Called(ctx, tx, userContent)
	return args.Error(0)
}

func (m *MockUserContentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.UserContent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserContent), args.Error(1)
}

func (m *MockUserContentRepository) GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.UserContent, error) {
	args := m.Called(ctx, userID, contentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserContent), args.Error(1)
}

func (m *MockUserContentRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.UserContent, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.UserContent), args.Error(1)
}

func (m *MockUserContentRepository) ListByUserWithFilter(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit, offset int) ([]*models.UserContent, error) {
	args := m.Called(ctx, userID, status, isFavorite, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.UserContent), args.Error(1)
}

func (m *MockUserContentRepository) UpdateMetadata(ctx context.Context, id uuid.UUID, status *string, scrollPosition *int, isFavorite *bool) error {
	args := m.Called(ctx, id, status, scrollPosition, isFavorite)
	return args.Error(0)
}

func (m *MockUserContentRepository) Delete(ctx context.Context, userID, contentID uuid.UUID) error {
	args := m.Called(ctx, userID, contentID)
	return args.Error(0)
}

func (m *MockUserContentRepository) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID, contentID uuid.UUID) error {
	args := m.Called(ctx, tx, userID, contentID)
	return args.Error(0)
}

func (m *MockUserContentRepository) Update(ctx context.Context, userContent *models.UserContent) error {
	args := m.Called(ctx, userContent)
	return args.Error(0)
}

func (m *MockUserContentRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, userContent *models.UserContent) error {
	args := m.Called(ctx, tx, userContent)
	return args.Error(0)
}

func (m *MockUserContentRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserContentRepository) Search(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]*models.UserContent, error) {
	args := m.Called(ctx, userID, query, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.UserContent), args.Error(1)
}

// MockContentRepository is a mock implementation of repository.ContentRepository
type MockContentRepository struct {
	mock.Mock
}

func (m *MockContentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentRepository) Create(ctx context.Context, content *models.Content) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockContentRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	args := m.Called(ctx, tx, content)
	return args.Error(0)
}

func (m *MockContentRepository) GetByContentHashAndFeedID(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	args := m.Called(ctx, contentHash, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentRepository) GetByContentHashesAndFeedID(ctx context.Context, hashes []string, feedID uuid.UUID) (map[string]*models.Content, error) {
	args := m.Called(ctx, hashes, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.Content), args.Error(1)
}

func (m *MockContentRepository) Update(ctx context.Context, content *models.Content) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockContentRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	args := m.Called(ctx, tx, content)
	return args.Error(0)
}

func (m *MockContentRepository) List(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Content), args.Error(1)
}

func (m *MockContentRepository) BulkCreate(ctx context.Context, contents []*models.Content) error {
	args := m.Called(ctx, contents)
	return args.Error(0)
}

func (m *MockContentRepository) DeleteOrphaned(ctx context.Context, olderThan time.Duration) (int64, error) {
	args := m.Called(ctx, olderThan)
	return args.Get(0).(int64), args.Error(1)
}

// TestListUserContents_Success tests successful listing of user contents
func TestListUserContents_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	userContents := []*models.UserContent{
		{
			ID:             uuid.New(),
			UserID:         userID,
			ContentID:      contentID,
			Status:         "unread",
			ScrollPosition: 0,
			IsFavorite:     false,
			AddedAt:        time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	content := &models.Content{
		ID:          contentID,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		ContentHash: "hash123",
		CleanedHTML: "<p>Test</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockUserContentRepo.On("ListByUserWithFilter", mock.Anything, userID, (*string)(nil), (*bool)(nil), 20, 0).
		Return(userContents, nil)
	mockUserContentRepo.On("CountByUser", mock.Anything, userID).Return(int64(1), nil)
	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(content, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, float64(1), response["total_count"])
	assert.Equal(t, float64(20), response["limit"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestListUserContents_WithFilters tests listing with status and favorite filters
func TestListUserContents_WithFilters(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	status := "read"
	isFavorite := true

	mockUserContentRepo.On("ListByUserWithFilter", mock.Anything, userID, &status, &isFavorite, 20, 0).
		Return([]*models.UserContent{}, nil)
	mockUserContentRepo.On("CountByUser", mock.Anything, userID).Return(int64(0), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents?status=read&is_favorite=true", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUserContentRepo.AssertExpectations(t)
}

// TestListUserContents_WithPagination tests pagination parameters
func TestListUserContents_WithPagination(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()

	mockUserContentRepo.On("ListByUserWithFilter", mock.Anything, userID, (*string)(nil), (*bool)(nil), 50, 10).
		Return([]*models.UserContent{}, nil)
	mockUserContentRepo.On("CountByUser", mock.Anything, userID).Return(int64(100), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents?limit=50&offset=10", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, float64(50), response["limit"])
	assert.Equal(t, float64(10), response["offset"])
	assert.NotNil(t, response["next_cursor"]) // Should have next cursor since 10+50 < 100
	mockUserContentRepo.AssertExpectations(t)
}

// TestListUserContents_InvalidUserID tests handling of invalid user ID
func TestListUserContents_InvalidUserID(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid/contents", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_user_id", response["error"])
}

// TestListUserContents_InvalidStatus tests handling of invalid status filter
func TestListUserContents_InvalidStatus(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents?status=invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_status", response["error"])
}

// TestAddContentToUser_Success tests successfully adding content to user
func TestAddContentToUser_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	content := &models.Content{
		ID:          contentID,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		ContentHash: "hash123",
		CleanedHTML: "<p>Test</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(content, nil).Twice()
	mockUserContentRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).Return(nil, nil)
	mockUserContentRepo.On("Create", mock.Anything, mock.MatchedBy(func(uc *models.UserContent) bool {
		return uc.UserID == userID && uc.ContentID == contentID && uc.Status == "unread"
	})).Return(nil)

	reqBody := map[string]interface{}{
		"content_id": contentID.String(),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, userID.String(), response["user_id"])
	assert.Equal(t, contentID.String(), response["content_id"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestAddContentToUser_DuplicatePrevention tests duplicate prevention
func TestAddContentToUser_DuplicatePrevention(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	content := &models.Content{
		ID: contentID,
	}

	existingUserContent := &models.UserContent{
		ID:        uuid.New(),
		UserID:    userID,
		ContentID: contentID,
	}

	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(content, nil)
	mockUserContentRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).Return(existingUserContent, nil)

	reqBody := map[string]interface{}{
		"content_id": contentID.String(),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "already_exists", response["error"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestAddContentToUser_ContentNotFound tests handling when content doesn't exist
func TestAddContentToUser_ContentNotFound(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(nil, sql.ErrNoRows)

	reqBody := map[string]interface{}{
		"content_id": contentID.String(),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "content_not_found", response["error"])
	mockContentRepo.AssertExpectations(t)
}

// TestUpdateUserContent_Success tests successful metadata update
func TestUpdateUserContent_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()
	ucID := uuid.New()

	existingUC := &models.UserContent{
		ID:        ucID,
		UserID:    userID,
		ContentID: contentID,
		Status:    "unread",
	}

	updatedUC := &models.UserContent{
		ID:             ucID,
		UserID:         userID,
		ContentID:      contentID,
		Status:         "read",
		ScrollPosition: 100,
		IsFavorite:     true,
		AddedAt:        time.Now(),
		UpdatedAt:      time.Now(),
	}

	content := &models.Content{
		ID:          contentID,
		Title:       "Test",
		OriginalURL: "https://example.com",
		ContentHash: "hash",
		CleanedHTML: "<p>Test</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	newStatus := "read"
	newScroll := 100
	newFavorite := true

	mockUserContentRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).Return(existingUC, nil)
	mockUserContentRepo.On("UpdateMetadata", mock.Anything, ucID, &newStatus, &newScroll, &newFavorite).Return(nil)
	mockUserContentRepo.On("GetByID", mock.Anything, ucID).Return(updatedUC, nil)
	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(content, nil)

	reqBody := map[string]interface{}{
		"status":          "read",
		"scroll_position": 100,
		"is_favorite":     true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "read", response["status"])
	assert.Equal(t, float64(100), response["scroll_position"])
	assert.Equal(t, true, response["is_favorite"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestUpdateUserContent_NotFound tests handling when user-content doesn't exist
func TestUpdateUserContent_NotFound(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).Return(nil, nil)

	reqBody := map[string]interface{}{
		"status": "read",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "not_found", response["error"])
	mockUserContentRepo.AssertExpectations(t)
}

// TestUpdateUserContent_InvalidStatus tests handling of invalid status
func TestUpdateUserContent_InvalidStatus(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	reqBody := map[string]interface{}{
		"status": "invalid_status",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_status", response["error"])
}

// TestDeleteUserContent_Success tests successful content deletion
func TestDeleteUserContent_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("Delete", mock.Anything, userID, contentID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.DeleteUserContent(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockUserContentRepo.AssertExpectations(t)
}

// TestDeleteUserContent_Error tests handling of deletion errors
func TestDeleteUserContent_Error(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("Delete", mock.Anything, userID, contentID).Return(errors.New("deletion failed"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.DeleteUserContent(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "delete_failed", response["error"])
	mockUserContentRepo.AssertExpectations(t)
}

// TestSearchUserContents_Success tests successful search
func TestSearchUserContents_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	userContents := []*models.UserContent{
		{
			ID:        uuid.New(),
			UserID:    userID,
			ContentID: contentID,
			Status:    "unread",
			AddedAt:   time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	content := &models.Content{
		ID:          contentID,
		Title:       "Search Result",
		OriginalURL: "https://example.com",
		ContentHash: "hash",
		CleanedHTML: "<p>Test</p>",
		SourceType:  "web",
		ImageURLs:   pq.StringArray{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockUserContentRepo.On("Search", mock.Anything, userID, "test query", 20, 0).Return(userContents, nil)
	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(content, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents/search?q=test+query", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.SearchUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Len(t, response, 1)
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestSearchUserContents_MissingQuery tests handling of missing search query
func TestSearchUserContents_MissingQuery(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents/search", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.SearchUserContents(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "missing_query", response["error"])
}
