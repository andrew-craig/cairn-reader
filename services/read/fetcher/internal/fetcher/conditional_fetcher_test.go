package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewConditionalFetcher tests creating a new conditional fetcher
func TestNewConditionalFetcher(t *testing.T) {
	httpClient := &http.Client{}
	fetcher := NewConditionalFetcher(httpClient)

	assert.NotNil(t, fetcher)
	assert.Equal(t, httpClient, fetcher.httpClient)
}

// TestFetchWithConditionals_NoHeaders tests fetch without conditional headers
func TestFetchWithConditionals_NoHeaders(t *testing.T) {
	expectedContent := []byte("Test content")
	lastModified := "Mon, 15 Jan 2024 10:00:00 GMT"
	etag := `"abc123"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no conditional headers are sent
		assert.Empty(t, r.Header.Get("If-Modified-Since"))
		assert.Empty(t, r.Header.Get("If-None-Match"))
		assert.Contains(t, r.Header.Get("User-Agent"), "Cairn-RSS-Fetcher")

		// Set response headers
		w.Header().Set("Last-Modified", lastModified)
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, expectedContent, result.Content)
	assert.NotNil(t, result.LastModified)
	assert.Equal(t, lastModified, *result.LastModified)
	assert.NotNil(t, result.ETag)
	assert.Equal(t, etag, *result.ETag)
}

// TestFetchWithConditionals_WithLastModified tests fetch with If-Modified-Since header
func TestFetchWithConditionals_WithLastModified(t *testing.T) {
	lastModifiedValue := "Mon, 15 Jan 2024 10:00:00 GMT"
	expectedContent := []byte("Updated content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify If-Modified-Since header is set
		assert.Equal(t, lastModifiedValue, r.Header.Get("If-Modified-Since"))

		// Simulate content has been modified
		w.Header().Set("Last-Modified", "Tue, 16 Jan 2024 10:00:00 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, &lastModifiedValue, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, expectedContent, result.Content)
}

// TestFetchWithConditionals_WithETag tests fetch with If-None-Match header
func TestFetchWithConditionals_WithETag(t *testing.T) {
	etagValue := `"abc123"`
	expectedContent := []byte("New content")
	newETag := `"def456"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify If-None-Match header is set
		assert.Equal(t, etagValue, r.Header.Get("If-None-Match"))

		// Simulate content has changed (new ETag)
		w.Header().Set("ETag", newETag)
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, &etagValue)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, expectedContent, result.Content)
	assert.NotNil(t, result.ETag)
	assert.Equal(t, newETag, *result.ETag)
}

// TestFetchWithConditionals_WithBothHeaders tests fetch with both conditional headers
func TestFetchWithConditionals_WithBothHeaders(t *testing.T) {
	lastModifiedValue := "Mon, 15 Jan 2024 10:00:00 GMT"
	etagValue := `"abc123"`
	expectedContent := []byte("Content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify both headers are set
		assert.Equal(t, lastModifiedValue, r.Header.Get("If-Modified-Since"))
		assert.Equal(t, etagValue, r.Header.Get("If-None-Match"))

		w.Header().Set("Last-Modified", "Tue, 16 Jan 2024 10:00:00 GMT")
		w.Header().Set("ETag", `"def456"`)
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, &lastModifiedValue, &etagValue)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, expectedContent, result.Content)
}

// TestFetchWithConditionals_NotModified tests 304 Not Modified response
func TestFetchWithConditionals_NotModified(t *testing.T) {
	lastModifiedValue := "Mon, 15 Jan 2024 10:00:00 GMT"
	etagValue := `"abc123"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify conditional headers
		assert.Equal(t, lastModifiedValue, r.Header.Get("If-Modified-Since"))
		assert.Equal(t, etagValue, r.Header.Get("If-None-Match"))

		// Return 304 Not Modified
		w.Header().Set("Last-Modified", lastModifiedValue)
		w.Header().Set("ETag", etagValue)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, &lastModifiedValue, &etagValue)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.NotModified)
	assert.Equal(t, http.StatusNotModified, result.StatusCode)
	assert.Nil(t, result.Content) // No content on 304
	// Headers should still be set
	assert.NotNil(t, result.LastModified)
	assert.Equal(t, lastModifiedValue, *result.LastModified)
	assert.NotNil(t, result.ETag)
	assert.Equal(t, etagValue, *result.ETag)
}

// TestFetchWithConditionals_NotModified_WithETagOnly tests 304 with only ETag
func TestFetchWithConditionals_NotModified_WithETagOnly(t *testing.T) {
	etagValue := `"xyz789"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, etagValue, r.Header.Get("If-None-Match"))

		w.Header().Set("ETag", etagValue)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, &etagValue)

	require.NoError(t, err)
	assert.True(t, result.NotModified)
	assert.Nil(t, result.Content)
}

// TestFetchWithConditionals_NotModified_WithLastModifiedOnly tests 304 with only Last-Modified
func TestFetchWithConditionals_NotModified_WithLastModifiedOnly(t *testing.T) {
	lastModifiedValue := "Wed, 17 Jan 2024 15:30:00 GMT"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, lastModifiedValue, r.Header.Get("If-Modified-Since"))

		w.Header().Set("Last-Modified", lastModifiedValue)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, &lastModifiedValue, nil)

	require.NoError(t, err)
	assert.True(t, result.NotModified)
	assert.Nil(t, result.Content)
}

// TestFetchWithConditionals_InvalidURL tests error handling for invalid URL
func TestFetchWithConditionals_InvalidURL(t *testing.T) {
	fetcher := NewConditionalFetcher(http.DefaultClient)
	result, err := fetcher.FetchWithConditionals(context.Background(), "://invalid-url", nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create request")
}

// TestFetchWithConditionals_HTTPError tests error handling for HTTP errors
func TestFetchWithConditionals_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

// TestFetchWithConditionals_NotFound tests 404 error handling
func TestFetchWithConditionals_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code: 404")
}

// TestFetchWithConditionals_ContextCancellation tests context cancellation
func TestFetchWithConditionals_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context to be cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := fetcher.FetchWithConditionals(ctx, server.URL, nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
}

// TestFetchWithConditionals_ContextTimeout tests context timeout
func TestFetchWithConditionals_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := fetcher.FetchWithConditionals(ctx, server.URL, nil, nil)

	require.Error(t, err)
	assert.Nil(t, result)
}

// TestFetchWithConditionals_EmptyHeaders tests with empty string headers
func TestFetchWithConditionals_EmptyHeaders(t *testing.T) {
	emptyString := ""
	expectedContent := []byte("Content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty headers should not be set
		assert.Empty(t, r.Header.Get("If-Modified-Since"))
		assert.Empty(t, r.Header.Get("If-None-Match"))

		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, &emptyString, &emptyString)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, expectedContent, result.Content)
}

// TestFetchWithConditionals_NoResponseHeaders tests when server doesn't return caching headers
func TestFetchWithConditionals_NoResponseHeaders(t *testing.T) {
	expectedContent := []byte("Content without headers")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't set Last-Modified or ETag headers
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, expectedContent, result.Content)
	assert.Nil(t, result.LastModified)
	assert.Nil(t, result.ETag)
}

// TestFetchWithConditionals_LargeContent tests fetching large content
func TestFetchWithConditionals_LargeContent(t *testing.T) {
	// Create 1MB of content
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(largeContent)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, largeContent, result.Content)
	assert.Len(t, result.Content, 1024*1024)
}

// TestFetchForItem_Success tests fetching for a feed item
func TestFetchForItem_Success(t *testing.T) {
	expectedContent := []byte("Article content")
	lastModified := "Mon, 15 Jan 2024 10:00:00 GMT"
	etag := `"item123"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers from item
		assert.Equal(t, lastModified, r.Header.Get("If-Modified-Since"))
		assert.Equal(t, etag, r.Header.Get("If-None-Match"))

		// Return new content
		newETag := `"item456"`
		w.Header().Set("ETag", newETag)
		w.Header().Set("Last-Modified", "Tue, 16 Jan 2024 10:00:00 GMT")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	feedItem := &models.FeedItem{
		ID:               uuid.New(),
		FeedID:           uuid.New(),
		ItemURL:          server.URL,
		HTTPLastModified: &lastModified,
		HTTPETag:         &etag,
	}

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchForItem(context.Background(), feedItem)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, expectedContent, result.Content)
}

// TestFetchForItem_NotModified tests 304 response for a feed item
func TestFetchForItem_NotModified(t *testing.T) {
	lastModified := "Mon, 15 Jan 2024 10:00:00 GMT"
	etag := `"unchanged"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, lastModified, r.Header.Get("If-Modified-Since"))
		assert.Equal(t, etag, r.Header.Get("If-None-Match"))

		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", lastModified)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	feedItem := &models.FeedItem{
		ID:               uuid.New(),
		FeedID:           uuid.New(),
		ItemURL:          server.URL,
		HTTPLastModified: &lastModified,
		HTTPETag:         &etag,
	}

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchForItem(context.Background(), feedItem)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.NotModified)
	assert.Nil(t, result.Content)
}

// TestFetchForItem_NoHeaders tests fetching item without cached headers
func TestFetchForItem_NoHeaders(t *testing.T) {
	expectedContent := []byte("Fresh content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No conditional headers should be sent
		assert.Empty(t, r.Header.Get("If-Modified-Since"))
		assert.Empty(t, r.Header.Get("If-None-Match"))

		w.Header().Set("ETag", `"new123"`)
		w.WriteHeader(http.StatusOK)
		w.Write(expectedContent)
	}))
	defer server.Close()

	feedItem := &models.FeedItem{
		ID:               uuid.New(),
		FeedID:           uuid.New(),
		ItemURL:          server.URL,
		HTTPLastModified: nil, // No cached headers
		HTTPETag:         nil,
	}

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchForItem(context.Background(), feedItem)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.NotModified)
	assert.Equal(t, expectedContent, result.Content)
	assert.NotNil(t, result.ETag)
}

// TestFetchForItem_Error tests error handling for feed item fetch
func TestFetchForItem_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	feedItem := &models.FeedItem{
		ID:      uuid.New(),
		FeedID:  uuid.New(),
		ItemURL: server.URL,
	}

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchForItem(context.Background(), feedItem)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code")
}

// TestFetchWithConditionals_ETagComparison tests content update detection via ETag
func TestFetchWithConditionals_ETagComparison(t *testing.T) {
	oldETag := `"v1"`
	newETag := `"v2"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, oldETag, r.Header.Get("If-None-Match"))

		// Return new ETag indicating content changed
		w.Header().Set("ETag", newETag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Updated content"))
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, &oldETag)

	require.NoError(t, err)
	assert.False(t, result.NotModified)
	assert.NotNil(t, result.ETag)
	assert.Equal(t, newETag, *result.ETag)
	// ETags are different, indicating content was updated
	assert.NotEqual(t, oldETag, *result.ETag)
}

// TestFetchWithConditionals_WeakETag tests handling of weak ETags
func TestFetchWithConditionals_WeakETag(t *testing.T) {
	weakETag := `W/"weak-etag"`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, weakETag, r.Header.Get("If-None-Match"))

		// Server can still return 304 with weak ETag
		w.Header().Set("ETag", weakETag)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, &weakETag)

	require.NoError(t, err)
	assert.True(t, result.NotModified)
	assert.NotNil(t, result.ETag)
	assert.Equal(t, weakETag, *result.ETag)
}

// TestFetchWithConditionals_UserAgent tests that user agent is set correctly
func TestFetchWithConditionals_UserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		assert.NotEmpty(t, userAgent)
		assert.Contains(t, userAgent, "Cairn-RSS-Fetcher/1.0")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	fetcher := NewConditionalFetcher(server.Client())
	result, err := fetcher.FetchWithConditionals(context.Background(), server.URL, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, result)
}
