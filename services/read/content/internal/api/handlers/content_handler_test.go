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
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockContentService is a mock implementation of service.ContentService
type MockContentService struct {
	mock.Mock
}

func (m *MockContentService) CreateFromHTML(ctx context.Context, url, html, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error) {
	args := m.Called(ctx, url, html, sourceType, sourceFeedID, publishedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentService) CreateFromURL(ctx context.Context, url, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error) {
	args := m.Called(ctx, url, sourceType, sourceFeedID, publishedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentService) UpdateContent(ctx context.Context, contentID uuid.UUID, url, html string, publishedAt *time.Time) (*models.Content, error) {
	args := m.Called(ctx, contentID, url, html, publishedAt)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentService) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentService) BulkCreateFromHTML(ctx context.Context, items []service.BulkContentItem) ([]*models.Content, []service.BulkCreateError, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		var emptyErrors []service.BulkCreateError
		if args.Get(1) != nil {
			emptyErrors = args.Get(1).([]service.BulkCreateError)
		}
		return nil, emptyErrors, args.Error(2)
	}
	var errors []service.BulkCreateError
	if args.Get(1) != nil {
		errors = args.Get(1).([]service.BulkCreateError)
	}
	return args.Get(0).([]*models.Content), errors, args.Error(2)
}

func (m *MockContentService) CheckDuplicates(ctx context.Context, items []service.DuplicateCheckItem) (map[string]*models.Content, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.Content), args.Error(1)
}

func (m *MockContentService) CheckDuplicate(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	args := m.Called(ctx, contentHash, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentService) ListContents(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Content), args.Error(1)
}

// TestCreateContent_Success tests successful content creation
func TestCreateContent_Success(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	expectedContent := &models.Content{
		ID:          contentID,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		ContentHash: "hash123",
		CleanedHTML: "<p>Test content</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	html := "<html><body>Test</body></html>"
	mockService.On("CreateFromHTML", mock.Anything, "https://example.com", html, "web", (*uuid.UUID)(nil), (*time.Time)(nil)).
		Return(expectedContent, nil)

	reqBody := map[string]interface{}{
		"url":         "https://example.com",
		"html":        html,
		"source_type": "web",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateContent(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, contentID.String(), response["id"])
	assert.Equal(t, "Test Article", response["title"])
	mockService.AssertExpectations(t)
}

// TestCreateContent_FromURL tests content creation from URL (no HTML provided)
func TestCreateContent_FromURL(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	expectedContent := &models.Content{
		ID:          contentID,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		ContentHash: "hash123",
		CleanedHTML: "<p>Test content</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockService.On("CreateFromURL", mock.Anything, "https://example.com", "web", (*uuid.UUID)(nil), (*time.Time)(nil)).
		Return(expectedContent, nil)

	reqBody := map[string]interface{}{
		"url":         "https://example.com",
		"source_type": "web",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateContent(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockService.AssertExpectations(t)
}

// TestCreateContent_InvalidJSON tests handling of invalid JSON
func TestCreateContent_InvalidJSON(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_request", response["error"])
}

// TestCreateContent_MissingURL tests validation when URL is missing
func TestCreateContent_MissingURL(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	reqBody := map[string]interface{}{
		"html":        "<html><body>Test</body></html>",
		"source_type": "web",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "URL is required")
}

// TestCreateContent_InvalidSourceType tests validation of invalid source type
func TestCreateContent_InvalidSourceType(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	reqBody := map[string]interface{}{
		"url":         "https://example.com",
		"source_type": "invalid",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "Invalid source_type")
}

// TestCreateContent_ServiceError tests handling of service errors
func TestCreateContent_ServiceError(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	mockService.On("CreateFromURL", mock.Anything, "https://example.com", "web", (*uuid.UUID)(nil), (*time.Time)(nil)).
		Return(nil, errors.New("service error"))

	reqBody := map[string]interface{}{
		"url":         "https://example.com",
		"source_type": "web",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateContent(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "creation_failed", response["error"])
	mockService.AssertExpectations(t)
}

// TestUpdateContent_Success tests successful content update
func TestUpdateContent_Success(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	expectedContent := &models.Content{
		ID:          contentID,
		Title:       "Updated Article",
		OriginalURL: "https://example.com",
		ContentHash: "newhash",
		CleanedHTML: "<p>Updated content</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockService.On("UpdateContent", mock.Anything, contentID, "https://example.com", "<p>Updated</p>", (*time.Time)(nil)).
		Return(expectedContent, nil)

	reqBody := map[string]interface{}{
		"url":  "https://example.com",
		"html": "<p>Updated</p>",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Set up chi router context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateContent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "Updated Article", response["title"])
	mockService.AssertExpectations(t)
}

// TestUpdateContent_InvalidID tests handling of invalid content ID
func TestUpdateContent_InvalidID(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	reqBody := map[string]interface{}{
		"url":  "https://example.com",
		"html": "<p>Updated</p>",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contents/invalid-uuid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_id", response["error"])
}

// TestUpdateContent_NotFound tests handling when content is not found
func TestUpdateContent_NotFound(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	mockService.On("UpdateContent", mock.Anything, contentID, "https://example.com", "<p>Updated</p>", (*time.Time)(nil)).
		Return(nil, sql.ErrNoRows)

	reqBody := map[string]interface{}{
		"url":  "https://example.com",
		"html": "<p>Updated</p>",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateContent(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "not_found", response["error"])
	mockService.AssertExpectations(t)
}

// TestUpdateContent_MissingFields tests validation when required fields are missing
func TestUpdateContent_MissingFields(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	reqBody := map[string]interface{}{
		"url": "https://example.com",
		// Missing html field
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contents/"+contentID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
}

// TestGetContent_Success tests successful content retrieval
func TestGetContent_Success(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	expectedContent := &models.Content{
		ID:          contentID,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		ContentHash: "hash123",
		CleanedHTML: "<p>Test content</p>",
		SourceType:  "web",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockService.On("GetByID", mock.Anything, contentID).Return(expectedContent, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/"+contentID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetContent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, contentID.String(), response["id"])
	assert.Equal(t, "Test Article", response["title"])
	mockService.AssertExpectations(t)
}

// TestGetContent_InvalidID tests handling of invalid UUID format
func TestGetContent_InvalidID(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/invalid-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_id", response["error"])
}

// TestGetContent_NotFound tests handling when content is not found
func TestGetContent_NotFound(t *testing.T) {
	mockService := new(MockContentService)
	handler := NewContentHandler(mockService)

	contentID := uuid.New()
	mockService.On("GetByID", mock.Anything, contentID).Return(nil, sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contents/"+contentID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("content_id", contentID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.GetContent(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "not_found", response["error"])
	mockService.AssertExpectations(t)
}
