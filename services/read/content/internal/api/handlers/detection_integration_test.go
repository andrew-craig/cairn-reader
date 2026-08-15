//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/stretchr/testify/assert"
)

// TestDetectURL_RealFeed tests detection against a real RSS feed
func TestDetectURL_RealFeed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("CAIRN_SKIP_NETWORK_TESTS") == "1" {
		t.Skip("Skipping network-dependent integration test (CAIRN_SKIP_NETWORK_TESTS=1)")
	}

	// Setup real detector (not a mock)
	detector := service.NewURLDetector()
	handler := NewDetectionHandler(detector)

	// Create request with a real RSS feed URL
	reqBody := dto.DetectURLRequest{
		URL: "https://blog.golang.org/feed.atom",
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

	// Should detect as feed
	assert.Equal(t, "feed", response.Type)
	assert.NotNil(t, response.Title)

	t.Logf("Detected feed: %s (title: %v)", response.URL, *response.Title)
}

// TestDetectURL_RealPage tests detection against a real HTML page
func TestDetectURL_RealPage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("CAIRN_SKIP_NETWORK_TESTS") == "1" {
		t.Skip("Skipping network-dependent integration test (CAIRN_SKIP_NETWORK_TESTS=1)")
	}

	// Setup real detector (not a mock)
	detector := service.NewURLDetector()
	handler := NewDetectionHandler(detector)

	// Create request with a real page URL
	reqBody := dto.DetectURLRequest{
		URL: "https://example.com",
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

	// Should detect as page
	assert.Equal(t, "page", response.Type)
	assert.NotNil(t, response.Title)

	t.Logf("Detected page: %s (title: %v)", response.URL, *response.Title)
}
