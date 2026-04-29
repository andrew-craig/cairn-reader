package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContentServiceClient(t *testing.T) {
	config := ContentServiceConfig{
		BaseURL: "http://localhost:8080",
	}

	client := NewContentServiceClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	assert.Equal(t, 3, client.maxRetries)
	assert.Equal(t, 1*time.Second, client.retryDelay)
	assert.NotNil(t, client.circuitBreaker)
}

func TestNewContentServiceClient_CustomConfig(t *testing.T) {
	config := ContentServiceConfig{
		BaseURL:    "http://localhost:9090",
		Timeout:    10 * time.Second,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	}

	client := NewContentServiceClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:9090", client.baseURL)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
	assert.Equal(t, 5, client.maxRetries)
	assert.Equal(t, 2*time.Second, client.retryDelay)
}

func TestCreateContent_Success(t *testing.T) {
	feedID := uuid.New()
	expectedResponse := ContentResponse{
		ID:           uuid.New(),
		ContentHash:  "abc123",
		CleanedHTML:  "<p>Test content</p>",
		OriginalURL:  "https://example.com/article",
		Title:        "Test Article",
		SourceType:   "rss",
		SourceFeedID: &feedID,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/content", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: server.URL,
	})

	htmlContent := "<html><body>Test</body></html>"
	req := CreateContentRequest{
		URL:          "https://example.com/article",
		HTML:         &htmlContent,
		SourceType:   "rss",
		SourceFeedID: &feedID,
	}

	result, err := client.CreateContent(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse.ID, result.ID)
	assert.Equal(t, expectedResponse.ContentHash, result.ContentHash)
	assert.Equal(t, expectedResponse.Title, result.Title)
}

func TestUpdateContent_Success(t *testing.T) {
	contentID := uuid.New()
	expectedResponse := ContentResponse{
		ID:          contentID,
		ContentHash: "updated123",
		CleanedHTML: "<p>Updated content</p>",
		OriginalURL: "https://example.com/article",
		Title:       "Updated Article",
		SourceType:  "rss",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/v1/content/"+contentID.String(), r.URL.Path)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: server.URL,
	})

	req := UpdateContentRequest{
		URL:  "https://example.com/article",
		HTML: "<html><body>Updated</body></html>",
	}

	result, err := client.UpdateContent(context.Background(), contentID, req)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse.ID, result.ID)
	assert.Equal(t, expectedResponse.ContentHash, result.ContentHash)
}

func TestBulkCreateContent_Success(t *testing.T) {
	feedID := uuid.New()
	expectedResponse := BulkCreateContentResponse{
		Created: []*ContentResponse{
			{
				ID:           uuid.New(),
				ContentHash:  "abc123",
				Title:        "Article 1",
				SourceFeedID: &feedID,
			},
			{
				ID:           uuid.New(),
				ContentHash:  "def456",
				Title:        "Article 2",
				SourceFeedID: &feedID,
			},
		},
		Existing: []*ContentResponse{},
		Failed:   []BulkFailedItem{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/content/bulk", r.URL.Path)

		var req BulkCreateContentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Len(t, req.Contents, 2)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: server.URL,
	})

	contents := []BulkContentItem{
		{
			URL:          "https://example.com/article1",
			HTML:         "<html><body>Article 1</body></html>",
			SourceType:   "rss",
			SourceFeedID: &feedID,
		},
		{
			URL:          "https://example.com/article2",
			HTML:         "<html><body>Article 2</body></html>",
			SourceType:   "rss",
			SourceFeedID: &feedID,
		},
	}

	result, err := client.BulkCreateContent(context.Background(), contents)

	require.NoError(t, err)
	assert.Len(t, result.Created, 2)
	assert.Len(t, result.Failed, 0)
}

func TestBulkCreateContent_ValidationErrors(t *testing.T) {
	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: "http://localhost:8080",
	})

	// Test empty array
	_, err := client.BulkCreateContent(context.Background(), []BulkContentItem{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Test too many items
	tooManyContents := make([]BulkContentItem, 101)
	for i := range tooManyContents {
		tooManyContents[i] = BulkContentItem{
			URL:  "https://example.com",
			HTML: "<html></html>",
		}
	}
	_, err = client.BulkCreateContent(context.Background(), tooManyContents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum 100 items")
}

func TestCheckDuplicates_Success(t *testing.T) {
	feedID := uuid.New()
	contentID := uuid.New()

	expectedResponse := CheckDuplicatesResponse{
		Results: []DuplicateCheckResult{
			{
				ContentHash: "abc123",
				Exists:      true,
				ContentID:   &contentID,
			},
			{
				ContentHash: "def456",
				Exists:      false,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/content/check-duplicate", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: server.URL,
	})

	items := []CheckDuplicatesItem{
		{
			ContentHash:  "abc123",
			SourceFeedID: feedID,
		},
		{
			ContentHash:  "def456",
			SourceFeedID: feedID,
		},
	}

	result, err := client.CheckDuplicates(context.Background(), items)

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.True(t, result.Results[0].Exists)
	assert.False(t, result.Results[1].Exists)
}

func TestAddContentToUsers_Success(t *testing.T) {
	contentID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	expectedResponse := BulkAddToUsersResponse{
		Succeeded: []uuid.UUID{uuid.New(), uuid.New()},
		Failed:    []BulkFailedItem{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/internal/content/user/bulk", r.URL.Path)

		var req BulkAddToUsersRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Len(t, req.Items, 2)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: server.URL,
	})

	items := []BulkAddToUsersItem{
		{
			ContentID: contentID,
			UserID:    userID1,
			Status:    "unread",
		},
		{
			ContentID: contentID,
			UserID:    userID2,
			Status:    "unread",
		},
	}

	result, err := client.AddContentToUsers(context.Background(), items)

	require.NoError(t, err)
	assert.Len(t, result.Succeeded, 2)
	assert.Len(t, result.Failed, 0)
}

func TestRetryLogic_SuccessOnRetry(t *testing.T) {
	attemptCount := 0
	expectedResponse := ContentResponse{
		ID:    uuid.New(),
		Title: "Test Article",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "server_error",
				Message: "Internal server error",
			})
			return
		}
		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL:    server.URL,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond, // Short delay for testing
	})

	htmlContent := "<html><body>Test</body></html>"
	req := CreateContentRequest{
		URL:        "https://example.com/article",
		HTML:       &htmlContent,
		SourceType: "rss",
	}

	result, err := client.CreateContent(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse.ID, result.ID)
	assert.Equal(t, 3, attemptCount)
}

func TestRetryLogic_MaxRetriesExceeded(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "server_error",
			Message: "Internal server error",
		})
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL:    server.URL,
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	})

	htmlContent := "<html><body>Test</body></html>"
	req := CreateContentRequest{
		URL:        "https://example.com/article",
		HTML:       &htmlContent,
		SourceType: "rss",
	}

	_, err := client.CreateContent(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")
	assert.Equal(t, 3, attemptCount) // Initial attempt + 2 retries
}

func TestRetryLogic_NonRetryableError(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid request",
		})
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL:    server.URL,
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	})

	htmlContent := "<html><body>Test</body></html>"
	req := CreateContentRequest{
		URL:        "https://example.com/article",
		HTML:       &htmlContent,
		SourceType: "rss",
	}

	_, err := client.CreateContent(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
	assert.Equal(t, 1, attemptCount) // Should not retry
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	htmlContent := "<html><body>Test</body></html>"
	req := CreateContentRequest{
		URL:        "https://example.com/article",
		HTML:       &htmlContent,
		SourceType: "rss",
	}

	_, err := client.CreateContent(ctx, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestHTTPErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "not_found",
			Message: "Content not found",
		})
	}))
	defer server.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL:    server.URL,
		MaxRetries: 0, // Disable retries for this test
	})

	contentID := uuid.New()
	req := UpdateContentRequest{
		URL:  "https://example.com/article",
		HTML: "<html><body>Test</body></html>",
	}

	_, err := client.UpdateContent(context.Background(), contentID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
	assert.Contains(t, err.Error(), "not_found")
}

func TestIsNonRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "HTTP 400 error",
			err:      assert.AnError,
			expected: false,
		},
		{
			name:     "HTTP 404 error",
			err:      &customError{msg: "HTTP 404: not found"},
			expected: true,
		},
		{
			name:     "validation error",
			err:      &customError{msg: "validation_error: invalid input"},
			expected: true,
		},
		{
			name:     "HTTP 500 error (retryable)",
			err:      &customError{msg: "HTTP 500: internal server error"},
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNonRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// customError is a simple error type for testing
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
