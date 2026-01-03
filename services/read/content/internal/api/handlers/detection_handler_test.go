package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn/services/read/content/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockURLDetector is a mock implementation of service.URLDetector
type MockURLDetector struct {
	mock.Mock
}

func (m *MockURLDetector) DetectURL(ctx context.Context, url string) (*service.URLDetectionResult, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.URLDetectionResult), args.Error(1)
}

// TestDetectURL_Feed tests successful feed detection
func TestDetectURL_Feed(t *testing.T) {
	// Setup mock
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	feedTitle := "Example Blog"
	mockDetector.On("DetectURL", mock.Anything, "https://example.com/feed.xml").
		Return(&service.URLDetectionResult{
			URL:   "https://example.com/feed.xml",
			Type:  service.URLTypeFeed,
			Title: &feedTitle,
		}, nil)

	// Create request
	reqBody := dto.DetectURLRequest{
		URL: "https://example.com/feed.xml",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DetectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.DetectURLResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/feed.xml", response.URL)
	assert.Equal(t, "feed", response.Type)
	assert.NotNil(t, response.Title)
	assert.Equal(t, "Example Blog", *response.Title)

	mockDetector.AssertExpectations(t)
}

// TestDetectURL_Page tests successful page detection
func TestDetectURL_Page(t *testing.T) {
	// Setup mock
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	pageTitle := "Example Article"
	mockDetector.On("DetectURL", mock.Anything, "https://example.com/article").
		Return(&service.URLDetectionResult{
			URL:   "https://example.com/article",
			Type:  service.URLTypePage,
			Title: &pageTitle,
		}, nil)

	// Create request
	reqBody := dto.DetectURLRequest{
		URL: "https://example.com/article",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DetectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.DetectURLResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/article", response.URL)
	assert.Equal(t, "page", response.Type)
	assert.NotNil(t, response.Title)
	assert.Equal(t, "Example Article", *response.Title)

	mockDetector.AssertExpectations(t)
}

// TestDetectURL_Unknown tests detection timeout/error handling
func TestDetectURL_Unknown(t *testing.T) {
	// Setup mock
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	mockDetector.On("DetectURL", mock.Anything, "https://example.com/timeout").
		Return(&service.URLDetectionResult{
			URL:   "https://example.com/timeout",
			Type:  service.URLTypeUnknown,
			Title: nil,
		}, nil)

	// Create request
	reqBody := dto.DetectURLRequest{
		URL: "https://example.com/timeout",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DetectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.DetectURLResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/timeout", response.URL)
	assert.Equal(t, "unknown", response.Type)
	assert.Nil(t, response.Title)

	mockDetector.AssertExpectations(t)
}

// TestDetectURL_InvalidRequest tests invalid request body
func TestDetectURL_InvalidRequest(t *testing.T) {
	// Setup mock
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	// Create invalid request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DetectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_request", response.Error)
}

// TestDetectURL_MissingURL tests missing URL validation
func TestDetectURL_MissingURL(t *testing.T) {
	// Setup mock
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	// Create request with empty URL
	reqBody := dto.DetectURLRequest{
		URL: "",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DetectURL(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response dto.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
	assert.Contains(t, response.Message, "URL is required")
}

// TestDetectURL_DetectorError tests error handling when detector fails
func TestDetectURL_DetectorError(t *testing.T) {
	// Setup mock
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	mockDetector.On("DetectURL", mock.Anything, "https://example.com/error").
		Return(nil, assert.AnError)

	// Create request
	reqBody := dto.DetectURLRequest{
		URL: "https://example.com/error",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute
	handler.DetectURL(w, req)

	// Assert - should return 200 with unknown type on error
	assert.Equal(t, http.StatusOK, w.Code)

	var response dto.DetectURLResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/error", response.URL)
	assert.Equal(t, "unknown", response.Type)
	assert.Nil(t, response.Title)

	mockDetector.AssertExpectations(t)
}
