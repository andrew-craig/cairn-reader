package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
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

func (m *MockURLDetector) DiscoverFeeds(ctx context.Context, pageURL string) ([]service.DiscoveredFeed, error) {
	args := m.Called(ctx, pageURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]service.DiscoveredFeed), args.Error(1)
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

	var envelope struct {
		Data dto.DetectURLResponse `json:"data"`
	}
	err := json.NewDecoder(w.Body).Decode(&envelope)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/feed.xml", envelope.Data.URL)
	assert.Equal(t, "feed", envelope.Data.Type)
	assert.NotNil(t, envelope.Data.Title)
	assert.Equal(t, "Example Blog", *envelope.Data.Title)

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

	var envelope struct {
		Data dto.DetectURLResponse `json:"data"`
	}
	err := json.NewDecoder(w.Body).Decode(&envelope)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/article", envelope.Data.URL)
	assert.Equal(t, "page", envelope.Data.Type)
	assert.NotNil(t, envelope.Data.Title)
	assert.Equal(t, "Example Article", *envelope.Data.Title)

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

	var envelope struct {
		Data dto.DetectURLResponse `json:"data"`
	}
	err := json.NewDecoder(w.Body).Decode(&envelope)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/timeout", envelope.Data.URL)
	assert.Equal(t, "unknown", envelope.Data.Type)
	assert.Nil(t, envelope.Data.Title)

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

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "bad_request", response["error"])
}

// TestDetectURL_BodyTooLarge tests that an oversized body is rejected with
// 413 before reaching the detector.
func TestDetectURL_BodyTooLarge(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	reqBody := dto.DetectURLRequest{
		URL: "https://example.com/" + strings.Repeat("a", maxSimpleRequestSize+1),
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/detect", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DetectURL(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	mockDetector.AssertNotCalled(t, "DetectURL")
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

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "URL is required")
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

	var envelope struct {
		Data dto.DetectURLResponse `json:"data"`
	}
	err := json.NewDecoder(w.Body).Decode(&envelope)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/error", envelope.Data.URL)
	assert.Equal(t, "unknown", envelope.Data.Type)
	assert.Nil(t, envelope.Data.Title)

	mockDetector.AssertExpectations(t)
}

// TestDiscoverFeed_Success tests successful feed discovery
func TestDiscoverFeed_Success(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	mockDetector.On("DiscoverFeeds", mock.Anything, "https://example.com").
		Return([]service.DiscoveredFeed{
			{URL: "https://example.com/feed.xml", Title: "Example Blog"},
			{URL: "https://example.com/atom.xml", Title: "Example Atom"},
		}, nil)

	reqBody := dto.DiscoverFeedRequest{URL: "https://example.com"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/discover-feed", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DiscoverFeed(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Data dto.DiscoverFeedResponse `json:"data"`
	}
	err := json.NewDecoder(w.Body).Decode(&envelope)
	assert.NoError(t, err)
	assert.Len(t, envelope.Data.Feeds, 2)
	assert.Equal(t, "https://example.com/feed.xml", envelope.Data.Feeds[0].URL)
	assert.Equal(t, "Example Blog", envelope.Data.Feeds[0].Title)
	assert.Equal(t, "https://example.com/atom.xml", envelope.Data.Feeds[1].URL)
	assert.Equal(t, "Example Atom", envelope.Data.Feeds[1].Title)

	mockDetector.AssertExpectations(t)
}

// TestDiscoverFeed_NoFeedsFound tests discovery with no feeds returns empty array
func TestDiscoverFeed_NoFeedsFound(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	mockDetector.On("DiscoverFeeds", mock.Anything, "https://example.com").
		Return([]service.DiscoveredFeed{}, nil)

	reqBody := dto.DiscoverFeedRequest{URL: "https://example.com"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/discover-feed", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DiscoverFeed(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the raw JSON contains an empty array (not null)
	assert.Contains(t, w.Body.String(), `"feeds":[]`)

	var envelope struct {
		Data dto.DiscoverFeedResponse `json:"data"`
	}
	err := json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&envelope)
	assert.NoError(t, err)
	assert.NotNil(t, envelope.Data.Feeds)
	assert.Len(t, envelope.Data.Feeds, 0)

	mockDetector.AssertExpectations(t)
}

// TestDiscoverFeed_BodyTooLarge tests that an oversized body is rejected
// with 413 before reaching the detector.
func TestDiscoverFeed_BodyTooLarge(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	reqBody := dto.DiscoverFeedRequest{
		URL: "https://example.com/" + strings.Repeat("a", maxSimpleRequestSize+1),
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/discover-feed", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DiscoverFeed(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	mockDetector.AssertNotCalled(t, "DiscoverFeeds")
}

// TestDiscoverFeed_MissingURL tests missing URL validation
func TestDiscoverFeed_MissingURL(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	reqBody := dto.DiscoverFeedRequest{URL: ""}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/discover-feed", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DiscoverFeed(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response["error"])
	assert.Contains(t, response["message"], "URL is required")
}

// TestDiscoverFeed_InvalidJSON tests invalid JSON returns 400
func TestDiscoverFeed_InvalidJSON(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/discover-feed", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DiscoverFeed(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "bad_request", response["error"])
}

// TestDiscoverFeed_DiscoveryError tests error handling when DiscoverFeeds returns an error
func TestDiscoverFeed_DiscoveryError(t *testing.T) {
	mockDetector := new(MockURLDetector)
	handler := NewDetectionHandler(mockDetector)

	mockDetector.On("DiscoverFeeds", mock.Anything, "https://example.com").
		Return(nil, assert.AnError)

	reqBody := dto.DiscoverFeedRequest{URL: "https://example.com"}
	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/discover-feed", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.DiscoverFeed(w, req)

	// Assert - errors are never surfaced to the client, should return empty feeds
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"feeds":[]`)

	mockDetector.AssertExpectations(t)
}
