package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/pkg/auth"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
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

func (m *MockUserContentRepository) UpdateMetadata(ctx context.Context, id uuid.UUID, status *string, scrollPosition *float64, isFavorite *bool) error {
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

func (m *MockUserContentRepository) ListByUserWithCursor(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]*models.UserContent, error) {
	args := m.Called(ctx, userID, status, isFavorite, limit, cursorTime, cursorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.UserContent), args.Error(1)
}

func (m *MockUserContentRepository) SearchWithCursor(ctx context.Context, userID uuid.UUID, query string, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]*models.UserContent, error) {
	args := m.Called(ctx, userID, query, limit, cursorTime, cursorID)
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

func (m *MockContentRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*models.Content, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*models.Content), args.Error(1)
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

func (m *MockContentRepository) GetByContentHashAndURL(ctx context.Context, contentHash string, originalURL string) (*models.Content, error) {
	args := m.Called(ctx, contentHash, originalURL)
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

func (m *MockContentRepository) BulkCreate(ctx context.Context, contents []*models.Content) error {
	args := m.Called(ctx, contents)
	return args.Error(0)
}

func (m *MockContentRepository) DeleteOrphaned(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Get(0).(int64), args.Error(1)
}

// TestListUserContents_Success tests successful listing of user contents
func TestListUserContents_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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

	contentMap := map[uuid.UUID]*models.Content{contentID: content}

	// Handler fetches limit+1 (21) to detect has_more; only 1 item returned so has_more=false.
	mockUserContentRepo.On("ListByUserWithCursor", mock.Anything, userID, (*string)(nil), (*bool)(nil), 21, (*time.Time)(nil), (*uuid.UUID)(nil)).
		Return(userContents, nil)
	mockContentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(contentMap, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(20), pagination["limit"])
	assert.Equal(t, false, pagination["has_more"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestListUserContents_WithFilters tests listing with status and favorite filters
func TestListUserContents_WithFilters(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()
	status := "completed"
	isFavorite := true

	mockUserContentRepo.On("ListByUserWithCursor", mock.Anything, userID, &status, &isFavorite, 21, (*time.Time)(nil), (*uuid.UUID)(nil)).
		Return([]*models.UserContent{}, nil)
	mockContentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID]*models.Content{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents?status=completed&is_favorite=true", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUserContentRepo.AssertExpectations(t)
}

// TestListUserContents_WithPagination tests cursor-based pagination
func TestListUserContents_WithPagination(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()
	contentID := uuid.New()

	// Return limit+1 items so handler sets has_more=true.
	addedAt := time.Now()
	ucID := uuid.New()
	items := make([]*models.UserContent, 51)
	for i := range items {
		items[i] = &models.UserContent{
			ID:        uuid.New(),
			UserID:    userID,
			ContentID: contentID,
			Status:    "unread",
			AddedAt:   addedAt,
			UpdatedAt: addedAt,
		}
	}
	items[49].ID = ucID // last item in the page (index 49)

	pagedContent := &models.Content{
		ID:          contentID,
		Title:       "T",
		OriginalURL: "https://example.com",
		ContentHash: "h",
		CleanedHTML: "<p>T</p>",
		SourceType:  "web",
		CreatedAt:   addedAt,
		UpdatedAt:   addedAt,
	}
	pagedContentMap := map[uuid.UUID]*models.Content{contentID: pagedContent}

	mockUserContentRepo.On("ListByUserWithCursor", mock.Anything, userID, (*string)(nil), (*bool)(nil), 51, (*time.Time)(nil), (*uuid.UUID)(nil)).
		Return(items, nil)
	mockContentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(pagedContentMap, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents?limit=50", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()
	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(50), pagination["limit"])
	assert.Equal(t, true, pagination["has_more"])
	assert.NotEmpty(t, pagination["cursor"])
	// offset and total should not be present
	_, hasOffset := pagination["offset"]
	assert.False(t, hasOffset)
	_, hasTotal := pagination["total"]
	assert.False(t, hasTotal)
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestListUserContents_InvalidUserID tests handling of invalid user ID
func TestListUserContents_InvalidUserID(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	authUserID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid/contents", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, authUserID)

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "bad_request", response["error"])
}

// TestListUserContents_InvalidStatus tests handling of invalid status filter
func TestListUserContents_InvalidStatus(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents?status=invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
}

// TestAddContentToUser_Success tests successfully adding content to user
func TestAddContentToUser_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, userID.String(), data["user_id"])
	assert.Equal(t, contentID.String(), data["content_id"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestAddContentToUser_BodyTooLarge tests that an oversized body is rejected
// with 413 before reaching the repository/service layers.
func TestAddContentToUser_BodyTooLarge(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"title": strings.Repeat("a", maxSimpleRequestSize+1),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	mockUserContentRepo.AssertNotCalled(t, "Create")
}

// TestAddContentToUser_MalformedURLWithContentID tests that a malformed URL is
// rejected even when a valid content_id is also present. The handler prefers
// the URL flow whenever URL is non-empty (regardless of content_id), so
// validation must reject a bad URL on that path too.
func TestAddContentToUser_MalformedURLWithContentID(t *testing.T) {
	handler := NewUserContentHandler(
		&mockUserContentRepo{},
		&mockContentRepo{},
		&mockContentService{},
		&mockURLDetector{detectionType: service.URLTypePage},
		nil, // ingestRSSClient not needed for page submissions
	)

	userID := uuid.New()
	contentID := uuid.New()

	reqBody := map[string]interface{}{
		"url":        "not-a-url",
		"content_id": contentID.String(),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAddContentToUser_DuplicatePrevention tests duplicate prevention
func TestAddContentToUser_DuplicatePrevention(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "conflict", response["error"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestAddContentToUser_ContentNotFound tests handling when content doesn't exist
func TestAddContentToUser_ContentNotFound(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "not_found", response["error"])
	mockContentRepo.AssertExpectations(t)
}

// TestAddContentToUser_FeedAlreadySubscribed reproduces the 409->500
// mistranslation end-to-end: a real Ingest RSS subscribe handler response for
// "already subscribed" must surface to the app as 409, not 500. Points the
// real IngestRSSClient at an httptest server reproducing the real handler's
// wire response (services/read/fetcher/internal/api/handlers/subscription_handler.go).
func TestAddContentToUser_FeedAlreadySubscribed(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)

	fetcherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "user is already subscribed to this feed", nil, "v1")
	}))
	defer fetcherServer.Close()

	ingestRSSClient := service.NewIngestRSSClient(fetcherServer.URL, "test-key")
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, ingestRSSClient)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"url":  "https://example.com/feed.xml",
		"type": "feed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "conflict", response["error"])
}

// TestAddContentToUser_FeedLimitReached covers the 400 branch of the same
// fix: asserts the status is 400 (not 500) and that the client-facing
// message is the server's original text, not the sentinel-wrapped chain
// ("invalid feed subscription request: <message>").
func TestAddContentToUser_FeedLimitReached(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)

	fetcherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "user has reached maximum feed limit of 100", nil, "v1")
	}))
	defer fetcherServer.Close()

	ingestRSSClient := service.NewIngestRSSClient(fetcherServer.URL, "test-key")
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, ingestRSSClient)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"url":  "https://example.com/feed.xml",
		"type": "feed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "bad_request", response["error"])
	assert.Equal(t, "user has reached maximum feed limit of 100", response["message"])
}

// TestUpdateUserContent_Success tests successful metadata update
func TestUpdateUserContent_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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
		Status:         "completed",
		ScrollPosition: 0.5,
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

	newStatus := "completed"
	newScroll := 0.5
	newFavorite := true

	mockUserContentRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).Return(existingUC, nil)
	mockUserContentRepo.On("UpdateMetadata", mock.Anything, ucID, &newStatus, &newScroll, &newFavorite).Return(nil)
	mockUserContentRepo.On("GetByID", mock.Anything, ucID).Return(updatedUC, nil)
	mockContentRepo.On("GetByID", mock.Anything, contentID).Return(content, nil)

	reqBody := map[string]interface{}{
		"status":          "completed",
		"scroll_position": 0.5,
		"is_favorite":     true,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "completed", data["status"])
	assert.Equal(t, float64(0.5), data["scroll_position"])
	assert.Equal(t, true, data["is_favorite"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestUpdateUserContent_BodyTooLarge tests that an oversized body is
// rejected with 413 before reaching the repository layer.
func TestUpdateUserContent_BodyTooLarge(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()
	contentID := uuid.New()

	reqBody := map[string]interface{}{
		"status": strings.Repeat("a", maxSimpleRequestSize+1),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	mockUserContentRepo.AssertNotCalled(t, "UpdateMetadata")
}

// TestUpdateUserContent_NotFound tests handling when user-content doesn't exist
func TestUpdateUserContent_NotFound(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).Return(nil, nil)

	reqBody := map[string]interface{}{
		"status": "completed",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

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
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
}

// TestDeleteUserContent_Success tests successful content deletion
func TestDeleteUserContent_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("Delete", mock.Anything, userID, contentID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.DeleteUserContent(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockUserContentRepo.AssertExpectations(t)
}

// TestDeleteUserContent_Error tests handling of deletion errors
func TestDeleteUserContent_Error(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("Delete", mock.Anything, userID, contentID).Return(errors.New("deletion failed"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/contents/"+contentID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.DeleteUserContent(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "internal_error", response["error"])
	mockUserContentRepo.AssertExpectations(t)
}

// TestSearchUserContents_Success tests successful search
func TestSearchUserContents_Success(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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

	searchContentMap := map[uuid.UUID]*models.Content{contentID: content}

	// Handler fetches limit+1 (21); only 1 returned so has_more=false.
	mockUserContentRepo.On("SearchWithCursor", mock.Anything, userID, "test query", 21, (*time.Time)(nil), (*uuid.UUID)(nil)).Return(userContents, nil)
	mockContentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(searchContentMap, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents/search?q=test+query", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.SearchUserContents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	data := response["data"].([]interface{})
	assert.Len(t, data, 1)
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, false, pagination["has_more"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestSearchUserContents_MissingQuery tests handling of missing search query
func TestSearchUserContents_MissingQuery(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents/search", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = addAuthContextToRequest(req, userID)

	w := httptest.NewRecorder()

	handler.SearchUserContents(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "bad_request", response["error"])
}

// Helper function to add authenticated user ID to request context
func addAuthContextToRequest(req *http.Request, userID uuid.UUID) *http.Request {
	ctx := req.Context()
	// Add the authenticated user ID to context (as middleware would do)
	ctx = context.WithValue(ctx, auth.UserIDContextKey, userID)
	return req.WithContext(ctx)
}

// Helper function to add both route context and auth context
func setupUserContentRequest(userID uuid.UUID, authUserID uuid.UUID, path string, method string) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	// Add route context with user_id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Add auth context
	req = addAuthContextToRequest(req, authUserID)
	return req
}

// ============================================================================
// Authentication and Authorization Tests
// ============================================================================

// TestListUserContents_MissingAuth tests that missing authentication returns an error
func TestListUserContents_MissingAuth(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	userID := uuid.New()

	// Request without auth context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/contents", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	// NOTE: No auth context added - in production, middleware.RequireAuth would return 401

	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	// GetUserIDOrError returns error when auth context is missing
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "internal_error", response["error"])
}

// TestListUserContents_UnauthorizedUserAccess tests that user cannot access another user's content
func TestListUserContents_UnauthorizedUserAccess(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)

	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	requestedUserID := uuid.New()
	authenticatedUserID := uuid.New() // Different user

	req := setupUserContentRequest(requestedUserID, authenticatedUserID, "/api/v1/users/"+requestedUserID.String()+"/contents", http.MethodGet)
	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	// Should return 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "forbidden", response["error"])
	assert.Contains(t, response["message"], "own content")
	// Repository should not be called for unauthorized request
	mockUserContentRepo.AssertNotCalled(t, "ListByUserWithCursor")
}

// TestListUserContents_AuthorizedUserAccess tests that authenticated user can access their own content
func TestListUserContents_AuthorizedUserAccess(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

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
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		ContentHash: "hash123",
		CleanedHTML: "<p>Test</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	authContentMap := map[uuid.UUID]*models.Content{contentID: content}

	mockUserContentRepo.On("ListByUserWithCursor", mock.Anything, userID, (*string)(nil), (*bool)(nil), 21, (*time.Time)(nil), (*uuid.UUID)(nil)).
		Return(userContents, nil)
	mockContentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(authContentMap, nil)

	// Same user ID for both authenticated and requested user
	req := setupUserContentRequest(userID, userID, "/api/v1/users/"+userID.String()+"/contents", http.MethodGet)
	w := httptest.NewRecorder()

	handler.ListUserContents(w, req)

	// Should succeed
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(20), pagination["limit"])
	assert.Equal(t, false, pagination["has_more"])
	mockUserContentRepo.AssertExpectations(t)
	mockContentRepo.AssertExpectations(t)
}

// TestAddContentToUser_UnauthorizedUserAccess tests that user cannot add content to another user's account
func TestAddContentToUser_UnauthorizedUserAccess(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	requestedUserID := uuid.New()
	authenticatedUserID := uuid.New() // Different user

	req := setupUserContentRequest(requestedUserID, authenticatedUserID, "/api/v1/users/"+requestedUserID.String()+"/contents", http.MethodPost)
	w := httptest.NewRecorder()

	handler.AddContentToUser(w, req)

	// Should return 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "forbidden", response["error"])
	mockUserContentRepo.AssertNotCalled(t, "Create")
}

// TestUpdateUserContent_UnauthorizedUserAccess tests that user cannot update another user's content
func TestUpdateUserContent_UnauthorizedUserAccess(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	requestedUserID := uuid.New()
	authenticatedUserID := uuid.New() // Different user
	contentID := uuid.New()

	path := "/api/v1/users/" + requestedUserID.String() + "/contents/" + contentID.String()
	req := httptest.NewRequest(http.MethodPatch, path, nil)

	// Add route context with user_id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", requestedUserID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Add auth context
	req = addAuthContextToRequest(req, authenticatedUserID)

	w := httptest.NewRecorder()

	handler.UpdateUserContent(w, req)

	// Should return 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "forbidden", response["error"])
	mockUserContentRepo.AssertNotCalled(t, "UpdateMetadata")
}

// TestDeleteUserContent_UnauthorizedUserAccess tests that user cannot delete another user's content
func TestDeleteUserContent_UnauthorizedUserAccess(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	requestedUserID := uuid.New()
	authenticatedUserID := uuid.New() // Different user
	contentID := uuid.New()

	path := "/api/v1/users/" + requestedUserID.String() + "/contents/" + contentID.String()
	req := httptest.NewRequest(http.MethodDelete, path, nil)

	// Add route context with user_id parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", requestedUserID.String())
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Add auth context
	req = addAuthContextToRequest(req, authenticatedUserID)

	w := httptest.NewRecorder()

	handler.DeleteUserContent(w, req)

	// Should return 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "forbidden", response["error"])
	mockUserContentRepo.AssertNotCalled(t, "Delete")
}

// TestSearchUserContents_UnauthorizedUserAccess tests that user cannot search another user's content
func TestSearchUserContents_UnauthorizedUserAccess(t *testing.T) {
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewUserContentHandler(mockUserContentRepo, mockContentRepo, nil, nil, nil)

	requestedUserID := uuid.New()
	authenticatedUserID := uuid.New() // Different user

	req := setupUserContentRequest(requestedUserID, authenticatedUserID, "/api/v1/users/"+requestedUserID.String()+"/contents/search?q=test", http.MethodGet)
	w := httptest.NewRecorder()

	handler.SearchUserContents(w, req)

	// Should return 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "forbidden", response["error"])
	mockUserContentRepo.AssertNotCalled(t, "Search")
}
