package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestBulkCreateContent_Success tests successful bulk content creation
func TestBulkCreateContent_Success(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	content1 := &models.Content{
		ID:          uuid.New(),
		Title:       "Article 1",
		OriginalURL: "https://example.com/1",
		ContentHash: "hash1",
		CleanedHTML: "<p>Content 1</p>",
		SourceType:  "rss",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	content2 := &models.Content{
		ID:          uuid.New(),
		Title:       "Article 2",
		OriginalURL: "https://example.com/2",
		ContentHash: "hash2",
		CleanedHTML: "<p>Content 2</p>",
		SourceType:  "rss",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	contents := []*models.Content{content1, content2}
	var bulkErrors []service.BulkCreateError

	mockService.On("BulkCreateFromHTML", mock.Anything, mock.Anything).
		Return(contents, bulkErrors, nil)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"url":         "https://example.com/1",
				"html":        "<html><body>Content 1</body></html>",
				"source_type": "rss",
			},
			{
				"url":         "https://example.com/2",
				"html":        "<html><body>Content 2</body></html>",
				"source_type": "rss",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkCreateContent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var envelope map[string]interface{}
	json.NewDecoder(w.Body).Decode(&envelope)
	response := envelope["data"].(map[string]interface{})
	created := response["created"].([]interface{})
	assert.Len(t, created, 2)
	failed := response["failed"].([]interface{})
	assert.Len(t, failed, 0)
	mockService.AssertExpectations(t)
}

// TestBulkCreateContent_PartialFailure tests bulk creation with some failures
func TestBulkCreateContent_PartialFailure(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	content1 := &models.Content{
		ID:          uuid.New(),
		Title:       "Article 1",
		OriginalURL: "https://example.com/1",
		ContentHash: "hash1",
		CleanedHTML: "<p>Content 1</p>",
		SourceType:  "rss",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	contents := []*models.Content{content1}
	bulkErrors := []service.BulkCreateError{
		{
			Index:   1,
			URL:     "https://example.com/2",
			Message: "failed to process",
		},
	}

	mockService.On("BulkCreateFromHTML", mock.Anything, mock.Anything).
		Return(contents, bulkErrors, nil)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"url":         "https://example.com/1",
				"html":        "<html><body>Content 1</body></html>",
				"source_type": "rss",
			},
			{
				"url":         "https://example.com/2",
				"html":        "<html><body>Content 2</body></html>",
				"source_type": "rss",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkCreateContent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var envelope map[string]interface{}
	json.NewDecoder(w.Body).Decode(&envelope)
	response := envelope["data"].(map[string]interface{})
	created := response["created"].([]interface{})
	assert.Len(t, created, 1)
	failed := response["failed"].([]interface{})
	assert.Len(t, failed, 1)
	mockService.AssertExpectations(t)
}

// TestBulkCreateContent_EmptyArray tests validation of empty array
func TestBulkCreateContent_EmptyArray(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkCreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "contents")
}

// TestBulkCreateContent_ExceedsLimit tests max 100 items validation
func TestBulkCreateContent_ExceedsLimit(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	// Create 101 items
	items := make([]map[string]interface{}, 101)
	for i := 0; i < 101; i++ {
		items[i] = map[string]interface{}{
			"url":         "https://example.com/" + string(rune(i)),
			"html":        "<html><body>Test</body></html>",
			"source_type": "rss",
		}
	}

	reqBody := map[string]interface{}{
		"contents": items,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkCreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "1-100")
}

// TestBulkCreateContent_MissingRequiredFields tests validation of required fields
func TestBulkCreateContent_MissingRequiredFields(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"url": "https://example.com/1",
				// Missing html field
				"source_type": "rss",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkCreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "HTML is required")
}

// TestBulkCreateContent_InvalidSourceType tests validation of source type
func TestBulkCreateContent_InvalidSourceType(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"url":         "https://example.com/1",
				"html":        "<html><body>Test</body></html>",
				"source_type": "invalid",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkCreateContent(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "Invalid source_type")
}

// TestCheckDuplicates_Success tests successful duplicate checking
func TestCheckDuplicates_Success(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	feedID := uuid.New()
	contentID := uuid.New()

	existingContent := &models.Content{
		ID:          contentID,
		Title:       "Existing Article",
		ContentHash: "hash1",
		SourceType:  "rss",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	duplicatesMap := map[string]*models.Content{
		"hash1": existingContent,
	}

	mockService.On("CheckDuplicates", mock.Anything, mock.Anything).Return(duplicatesMap, nil)

	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"content_hash":   "hash1",
				"source_feed_id": feedID.String(),
			},
			{
				"content_hash":   "hash2",
				"source_feed_id": feedID.String(),
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/check-duplicates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CheckDuplicates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var envelope map[string]interface{}
	json.NewDecoder(w.Body).Decode(&envelope)
	data := envelope["data"].(map[string]interface{})
	results := data["results"].([]interface{})
	assert.Len(t, results, 2)

	// First result should exist
	result1 := results[0].(map[string]interface{})
	assert.Equal(t, "hash1", result1["content_hash"])
	assert.Equal(t, true, result1["exists"])
	assert.NotNil(t, result1["content_id"])

	// Second result should not exist
	result2 := results[1].(map[string]interface{})
	assert.Equal(t, "hash2", result2["content_hash"])
	assert.Equal(t, false, result2["exists"])

	mockService.AssertExpectations(t)
}

// TestCheckDuplicates_EmptyArray tests validation of empty array
func TestCheckDuplicates_EmptyArray(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/check-duplicates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CheckDuplicates(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
}

// TestCheckDuplicates_ExceedsLimit tests max 100 items validation
func TestCheckDuplicates_ExceedsLimit(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	feedID := uuid.New()
	items := make([]map[string]interface{}, 101)
	for i := 0; i < 101; i++ {
		items[i] = map[string]interface{}{
			"content_hash":   "hash" + string(rune(i)),
			"source_feed_id": feedID.String(),
		}
	}

	reqBody := map[string]interface{}{
		"items": items,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/check-duplicates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CheckDuplicates(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "1-100")
}

// TestCheckDuplicates_ServiceError tests handling of service errors
func TestCheckDuplicates_ServiceError(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	feedID := uuid.New()

	mockService.On("CheckDuplicates", mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"content_hash":   "hash1",
				"source_feed_id": feedID.String(),
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contents/check-duplicates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CheckDuplicates(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "internal_error", response["error"])
	mockService.AssertExpectations(t)
}

// TestBulkAddToUsers_Success tests successful bulk add to users
func TestBulkAddToUsers_Success(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	userID1 := uuid.New()
	userID2 := uuid.New()
	contentID := uuid.New()

	mockUserContentRepo.On("BulkCreate", mock.Anything, mock.MatchedBy(func(ucs []*models.UserContent) bool {
		return len(ucs) == 2 &&
			ucs[0].UserID == userID1 &&
			ucs[1].UserID == userID2 &&
			ucs[0].ContentID == contentID &&
			ucs[1].ContentID == contentID
	})).Return(nil)

	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"user_id":    userID1.String(),
				"content_id": contentID.String(),
				"status":     "unread",
			},
			{
				"user_id":    userID2.String(),
				"content_id": contentID.String(),
				"status":     "unread",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/content/user/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkAddToUsersInternal(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var envelope map[string]interface{}
	json.NewDecoder(w.Body).Decode(&envelope)
	response := envelope["data"].(map[string]interface{})
	succeeded := response["succeeded"].([]interface{})
	assert.Len(t, succeeded, 2)
	failed := response["failed"].([]interface{})
	assert.Len(t, failed, 0)
	mockUserContentRepo.AssertExpectations(t)
}

// TestBulkAddToUsers_EmptyArray tests validation of empty array
func TestBulkAddToUsers_EmptyArray(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/bulk/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkAddToUsersInternal(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
}

// TestBulkAddToUsers_ExceedsLimit tests max 100 items validation
func TestBulkAddToUsers_ExceedsLimit(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	items := make([]map[string]interface{}, 101)
	for i := 0; i < 101; i++ {
		items[i] = map[string]interface{}{
			"user_id":    uuid.New().String(),
			"content_id": uuid.New().String(),
		}
	}

	reqBody := map[string]interface{}{
		"items": items,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/bulk/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkAddToUsersInternal(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "1-100")
}

// TestBulkAddToUsers_InvalidStatus tests validation of status
func TestBulkAddToUsers_InvalidStatus(t *testing.T) {
	mockService := new(MockContentService)
	mockUserContentRepo := new(MockUserContentRepository)
	mockContentRepo := new(MockContentRepository)
	handler := NewBulkHandler(mockService, mockUserContentRepo, mockContentRepo)

	userID := uuid.New()
	contentID := uuid.New()

	reqBody := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"user_id":    userID.String(),
				"content_id": contentID.String(),
				"status":     "invalid_status",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/bulk/contents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.BulkAddToUsersInternal(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "Invalid status")
}
