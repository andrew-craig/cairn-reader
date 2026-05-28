package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPayload() []EmailContentItem {
	return []EmailContentItem{
		{
			UserID:     uuid.New(),
			URL:        "mailto:newsletter@example.com?subject=Weekly+Digest",
			Type:       "email",
			HTML:       "<p>Newsletter content</p>",
			Title:      "Weekly Digest",
			Author:     "Newsletter Author",
			SourceType: "email",
		},
	}
}

// writeCreateResponse writes a POST /api/v1/content/bulk success response.
func writeCreateResponse(w http.ResponseWriter, contentID uuid.UUID) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": createResponseData{
			Created: []createdItem{{ID: contentID}},
		},
		"meta": map[string]string{},
	})
}

// writeAddResponse writes a POST /api/v1/internal/content/user/bulk success response.
func writeAddResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{"succeeded": []string{}, "failed": nil},
		"meta": map[string]string{},
	})
}

// twoStepServer builds a test server that serves both delivery endpoints.
// createHandler handles POST /api/v1/content/bulk.
// addHandler handles POST /api/v1/internal/content/user/bulk.
func twoStepServer(t *testing.T, createHandler, addHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/content/bulk":
			createHandler(w, r)
		case "/api/v1/internal/content/user/bulk":
			addHandler(w, r)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestContentServiceClient_DeliverContent_Success(t *testing.T) {
	contentID := uuid.New()

	srv := twoStepServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			writeCreateResponse(w, contentID)
		},
		func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-key", r.Header.Get("X-Internal-API-Key"))
			writeAddResponse(w)
		},
	)
	defer srv.Close()

	c := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	id, err := c.DeliverContent(context.Background(), newTestPayload())
	require.NoError(t, err)
	assert.Equal(t, contentID, id)
}

func TestContentServiceClient_DeliverContent_ExistingItem(t *testing.T) {
	existingID := uuid.New()

	srv := twoStepServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": createResponseData{
					Existing: []createdItem{{ID: existingID}},
				},
				"meta": map[string]string{},
			})
		},
		func(w http.ResponseWriter, _ *http.Request) { writeAddResponse(w) },
	)
	defer srv.Close()

	c := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	id, err := c.DeliverContent(context.Background(), newTestPayload())
	require.NoError(t, err)
	assert.Equal(t, existingID, id)
}

func TestContentServiceClient_DeliverContent_EmptyPayload(t *testing.T) {
	c := NewContentServiceClient(ContentServiceConfig{BaseURL: "http://unused"})
	_, err := c.DeliverContent(context.Background(), nil)
	assert.Error(t, err)
}

func TestContentServiceClient_DeliverContent_MissingAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when API key is missing")
	}))
	defer srv.Close()

	c := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL})
	_, err := c.DeliverContent(context.Background(), newTestPayload())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal API key is required")
}

func TestContentServiceClient_DeliverContent_RetryOnServerError(t *testing.T) {
	var createCallCount int32
	contentID := uuid.New()

	srv := twoStepServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&createCallCount, 1)
			if count < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writeCreateResponse(w, contentID)
		},
		func(w http.ResponseWriter, _ *http.Request) { writeAddResponse(w) },
	)
	defer srv.Close()

	// Use very short delays for testing by patching retryDelays.
	original := retryDelays
	retryDelays = []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}
	defer func() { retryDelays = original }()

	c := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	id, err := c.DeliverContent(context.Background(), newTestPayload())
	require.NoError(t, err)
	assert.Equal(t, contentID, id)
	assert.Equal(t, int32(3), atomic.LoadInt32(&createCallCount))
}

func TestContentServiceClient_DeliverContent_NoRetryOn4xx(t *testing.T) {
	var callCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	_, err := c.DeliverContent(context.Background(), newTestPayload())
	assert.Error(t, err)
	// Should only be called once — no retry on 400
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestContentServiceClient_CircuitBreaker_OpensAfterConsecutiveFailures(t *testing.T) {
	var callCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	original := retryDelays
	retryDelays = []time.Duration{} // no retries so failures are fast
	defer func() { retryDelays = original }()

	c := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})

	// Drive 5 consecutive failures to open the circuit.
	for i := 0; i < 5; i++ {
		//nolint:errcheck
		c.DeliverContent(context.Background(), newTestPayload())
	}

	countAfterOpen := atomic.LoadInt32(&callCount)

	// Next call should be rejected by the open circuit without hitting the server.
	_, err := c.DeliverContent(context.Background(), newTestPayload())
	assert.Error(t, err)
	assert.Equal(t, countAfterOpen, atomic.LoadInt32(&callCount), "circuit breaker should prevent server call")
}

func TestContentServiceClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewContentServiceClient(ContentServiceConfig{
		BaseURL:        srv.URL,
		Timeout:        5 * time.Second,
		InternalAPIKey: "test-key",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.DeliverContent(ctx, newTestPayload())
	assert.Error(t, err)
}
