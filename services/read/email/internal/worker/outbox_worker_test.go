package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/client"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFullOutboxRepo implements repository.OutboxRepository for outbox worker tests.
type mockFullOutboxRepo struct {
	getPendingFunc      func(ctx context.Context, limit int) ([]*models.ContentOutbox, error)
	updateDeliveryFunc  func(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time) error
	updateRetryFunc     func(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt time.Time, lastError string) error
	releaseClaimFunc    func(ctx context.Context, id uuid.UUID) error
	deleteDeliveredFunc func(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)
}

func (m *mockFullOutboxRepo) Create(ctx context.Context, outbox *models.ContentOutbox) error {
	return nil
}
func (m *mockFullOutboxRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error) {
	return nil, nil
}
func (m *mockFullOutboxRepo) GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	if m.getPendingFunc != nil {
		return m.getPendingFunc(ctx, limit)
	}
	return nil, nil
}
func (m *mockFullOutboxRepo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time) error {
	if m.updateDeliveryFunc != nil {
		return m.updateDeliveryFunc(ctx, id, status, contentServiceID, deliveredAt)
	}
	return nil
}
func (m *mockFullOutboxRepo) UpdateRetryInfo(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt time.Time, lastError string) error {
	if m.updateRetryFunc != nil {
		return m.updateRetryFunc(ctx, id, retryCount, nextRetryAt, lastError)
	}
	return nil
}
func (m *mockFullOutboxRepo) ReleaseClaim(ctx context.Context, id uuid.UUID) error {
	if m.releaseClaimFunc != nil {
		return m.releaseClaimFunc(ctx, id)
	}
	return nil
}
func (m *mockFullOutboxRepo) DeleteDelivered(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	if m.deleteDeliveredFunc != nil {
		return m.deleteDeliveredFunc(ctx, olderThan, batchSize)
	}
	return 0, nil
}

func makeOutboxEntry() *models.ContentOutbox {
	return &models.ContentOutbox{
		ID:         uuid.New(),
		RawEmailID: uuid.New(),
		UserID:     uuid.New(),
		ContentPayload: map[string]interface{}{
			"url":         "email://abc123",
			"html":        "<p>hello</p>",
			"title":       "Test Email",
			"author":      "Author",
			"source_type": "email",
		},
		DeliveryStatus: models.DeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     6,
		NextRetryAt:    time.Now(),
	}
}

func makeOutboxEntryWithURL(url string) *models.ContentOutbox {
	entry := makeOutboxEntry()
	entry.ContentPayload["url"] = url
	return entry
}

// TestOutboxWorker_DeliverBatch_FailedEntryDoesNotBlockSubsequentEntries proves C5:
// a downstream failure on the first entry must not stall the rest of the batch behind
// the client's retry schedule. retryDelays is deliberately left at its real production
// values (1m/5m/15m/1h/4h/12h) — patching it to milliseconds would hide exactly the bug
// this test exists to catch. On unfixed code, entry 1's failure blocks inside
// ContentServiceClient.DeliverContent for a full minute before deliverBatch ever reaches
// entry 2, so this test times out.
func TestOutboxWorker_DeliverBatch_FailedEntryDoesNotBlockSubsequentEntries(t *testing.T) {
	entry1 := makeOutboxEntryWithURL("email://entry-1")
	entry2 := makeOutboxEntryWithURL("email://entry-2")

	entry2Attempted := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/content/bulk" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":{},"meta":{}}`)
			return
		}

		var body struct {
			Contents []struct {
				URL string `json:"url"`
			} `json:"contents"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotEmpty(t, body.Contents)

		if body.Contents[0].URL == "email://entry-1" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		select {
		case entry2Attempted <- struct{}{}:
		default:
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"created":[{"id":%q}],"existing":[],"failed":[]}}`, uuid.New())
	}))
	defer server.Close()

	repo := &mockFullOutboxRepo{
		getPendingFunc: func(_ context.Context, _ int) ([]*models.ContentOutbox, error) {
			return []*models.ContentOutbox{entry1, entry2}, nil
		},
	}

	cc := client.NewContentServiceClient(client.ContentServiceConfig{BaseURL: server.URL, InternalAPIKey: "test-key"})
	w := NewOutboxWorker(repo, cc, OutboxWorkerConfig{BatchSize: 10, PollInterval: time.Hour})

	batchDone := make(chan struct{})
	go func() {
		w.deliverBatch(context.Background())
		close(batchDone)
	}()

	select {
	case <-entry2Attempted:
	case <-time.After(2 * time.Second):
		t.Fatal("entry 2 was not attempted within the same batch pass — entry 1's failure blocked the batch")
	}

	select {
	case <-batchDone:
	case <-time.After(2 * time.Second):
		t.Fatal("deliverBatch did not return promptly after both entries were attempted")
	}
}

func TestOutboxWorker_DeliverEntry_Success(t *testing.T) {
	entry := makeOutboxEntry()
	contentID := uuid.New()

	var unexpectedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/content/bulk":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"data":{"created":[{"id":%q}],"existing":[],"failed":[]}}`, contentID)
		case "/api/v1/internal/content/user/bulk":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"data":{},"meta":{}}`)
		default:
			unexpectedPath = r.URL.Path
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var deliveredStatus models.DeliveryStatus
	var deliveredContentID *uuid.UUID
	updateDeliveryCalled := false

	repo := &mockFullOutboxRepo{
		updateDeliveryFunc: func(_ context.Context, id uuid.UUID, status models.DeliveryStatus, cid *uuid.UUID, deliveredAt *time.Time) error {
			updateDeliveryCalled = true
			assert.Equal(t, entry.ID, id)
			assert.NotNil(t, deliveredAt)
			deliveredStatus = status
			deliveredContentID = cid
			return nil
		},
	}

	cc := client.NewContentServiceClient(client.ContentServiceConfig{BaseURL: server.URL, InternalAPIKey: "test-key"})
	w := NewOutboxWorker(repo, cc, OutboxWorkerConfig{BatchSize: 10, PollInterval: time.Second})

	w.deliverEntry(context.Background(), entry)

	require.Empty(t, unexpectedPath, "unexpected request path")
	require.True(t, updateDeliveryCalled, "deliverEntry must call UpdateDeliveryStatus on success")
	assert.Equal(t, models.DeliveryStatusDelivered, deliveredStatus)
	require.NotNil(t, deliveredContentID)
	assert.Equal(t, contentID, *deliveredContentID)
}

func TestOutboxWorker_DeliverEntry_MissingURL(t *testing.T) {
	entry := makeOutboxEntry()
	delete(entry.ContentPayload, "url")

	var retryCount int
	repo := &mockFullOutboxRepo{
		updateRetryFunc: func(_ context.Context, _ uuid.UUID, count int, _ time.Time, _ string) error {
			retryCount = count
			return nil
		},
	}
	cc := client.NewContentServiceClient(client.ContentServiceConfig{BaseURL: "http://test"})

	w := NewOutboxWorker(repo, cc, OutboxWorkerConfig{BatchSize: 10, PollInterval: time.Second})
	w.deliverEntry(context.Background(), entry)

	assert.Equal(t, 1, retryCount)
}

func TestOutboxWorker_RecordFailure_BackoffIncreases(t *testing.T) {
	var nextRetries []time.Time
	entry := makeOutboxEntry()

	repo := &mockFullOutboxRepo{
		updateRetryFunc: func(_ context.Context, _ uuid.UUID, _ int, nextRetryAt time.Time, _ string) error {
			nextRetries = append(nextRetries, nextRetryAt)
			return nil
		},
	}
	cc := client.NewContentServiceClient(client.ContentServiceConfig{BaseURL: "http://test"})
	w := NewOutboxWorker(repo, cc, OutboxWorkerConfig{})

	err := errors.New("delivery failed")
	now := time.Now()

	for i := 0; i < 3; i++ {
		entry.RetryCount = i
		w.recordFailure(context.Background(), entry, err)
	}

	require.Len(t, nextRetries, 3)
	// Each subsequent retry should be further in the future
	assert.True(t, nextRetries[1].After(nextRetries[0].Add(now.Sub(now))), "retry 2 should be after retry 1")
	// First retry: ~1m from now, second: ~2m, third: ~4m
	delta1 := nextRetries[0].Sub(now)
	delta2 := nextRetries[1].Sub(now)
	assert.True(t, delta2 > delta1, "retry delay should increase")
}

func TestOutboxBackoff(t *testing.T) {
	cases := []struct {
		retry    int
		expected time.Duration
	}{
		{1, 1 * time.Minute},
		{2, 2 * time.Minute},
		{3, 4 * time.Minute},
		{4, 8 * time.Minute},
		{5, 16 * time.Minute},
		{6, 32 * time.Minute},
	}
	for _, tc := range cases {
		got := outboxBackoff(tc.retry)
		assert.Equal(t, tc.expected, got, "retry %d", tc.retry)
	}
}

func TestOutboxWorker_Start_StopsOnContextCancel(t *testing.T) {
	repo := &mockFullOutboxRepo{}
	cc := client.NewContentServiceClient(client.ContentServiceConfig{BaseURL: "http://test"})
	w := NewOutboxWorker(repo, cc, OutboxWorkerConfig{PollInterval: 100 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("outbox worker did not stop after context cancellation")
	}
}

func TestOutboxToContentItem_AllFields(t *testing.T) {
	userID := uuid.New()
	entry := &models.ContentOutbox{
		ID:     uuid.New(),
		UserID: userID,
		ContentPayload: map[string]interface{}{
			"url":         "email://test-id",
			"html":        "<p>body</p>",
			"title":       "My Title",
			"author":      "John Doe",
			"source_type": "email",
		},
	}

	item, err := outboxToContentItem(entry)
	require.NoError(t, err)
	assert.Equal(t, "email://test-id", item.URL)
	assert.Equal(t, "<p>body</p>", item.HTML)
	assert.Equal(t, "My Title", item.Title)
	assert.Equal(t, "John Doe", item.Author)
	assert.Equal(t, "email", item.SourceType)
	assert.Equal(t, "email", item.Type)
	assert.Equal(t, userID, item.UserID)
}
