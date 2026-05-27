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

func TestContentServiceClient_DeliverContent_Success(t *testing.T) {
	contentID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/internal/content/user/bulk", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := deliverResponse{
			Created: []createdItem{{ID: contentID}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	id, err := client.DeliverContent(context.Background(), newTestPayload())
	require.NoError(t, err)
	assert.Equal(t, contentID, id)
}

func TestContentServiceClient_DeliverContent_ExistingItem(t *testing.T) {
	existingID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := deliverResponse{
			Existing: []createdItem{{ID: existingID}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	id, err := client.DeliverContent(context.Background(), newTestPayload())
	require.NoError(t, err)
	assert.Equal(t, existingID, id)
}

func TestContentServiceClient_DeliverContent_EmptyPayload(t *testing.T) {
	client := NewContentServiceClient(ContentServiceConfig{BaseURL: "http://unused"})
	_, err := client.DeliverContent(context.Background(), nil)
	assert.Error(t, err)
}

func TestContentServiceClient_DeliverContent_MissingAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called when API key is missing")
	}))
	defer srv.Close()

	client := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL})
	_, err := client.DeliverContent(context.Background(), newTestPayload())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestContentServiceClient_DeliverContent_RetryOnServerError(t *testing.T) {
	var callCount int32
	contentID := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		resp := deliverResponse{Created: []createdItem{{ID: contentID}}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Use very short delays for testing by patching retryDelays.
	original := retryDelays
	retryDelays = []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}
	defer func() { retryDelays = original }()

	client := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	id, err := client.DeliverContent(context.Background(), newTestPayload())
	require.NoError(t, err)
	assert.Equal(t, contentID, id)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestContentServiceClient_DeliverContent_NoRetryOn4xx(t *testing.T) {
	var callCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})
	_, err := client.DeliverContent(context.Background(), newTestPayload())
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

	client := NewContentServiceClient(ContentServiceConfig{BaseURL: srv.URL, InternalAPIKey: "test-key"})

	// Drive 5 consecutive failures to open the circuit.
	for i := 0; i < 5; i++ {
		//nolint:errcheck
		client.DeliverContent(context.Background(), newTestPayload())
	}

	countAfterOpen := atomic.LoadInt32(&callCount)

	// Next call should be rejected by the open circuit without hitting the server.
	_, err := client.DeliverContent(context.Background(), newTestPayload())
	assert.Error(t, err)
	assert.Equal(t, countAfterOpen, atomic.LoadInt32(&callCount), "circuit breaker should prevent server call")
}

func TestContentServiceClient_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewContentServiceClient(ContentServiceConfig{
		BaseURL:        srv.URL,
		Timeout:        5 * time.Second,
		InternalAPIKey: "test-key",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.DeliverContent(ctx, newTestPayload())
	assert.Error(t, err)
}
