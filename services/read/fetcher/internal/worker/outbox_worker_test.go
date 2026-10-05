package worker

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxWorker_NewOutboxWorker_WithNilConfig(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil, nil)

	assert.NotNil(t, worker)
	assert.NotNil(t, worker.config)
	assert.Equal(t, 5, worker.config.WorkerCount)
	assert.Equal(t, 20, worker.config.BatchSize)
	assert.Equal(t, 10*time.Second, worker.config.PollInterval)
	assert.Equal(t, 6, worker.config.MaxRetries)
}

func TestOutboxWorker_NewOutboxWorker_WithCustomConfig(t *testing.T) {
	customConfig := &OutboxWorkerConfig{
		WorkerCount:  10,
		BatchSize:    50,
		PollInterval: 5 * time.Second,
		MaxRetries:   3,
	}

	worker := NewOutboxWorker(customConfig, nil, nil, nil)

	assert.NotNil(t, worker)
	assert.Equal(t, customConfig, worker.config)
	assert.Equal(t, 10, worker.config.WorkerCount)
	assert.Equal(t, 50, worker.config.BatchSize)
	assert.Equal(t, 5*time.Second, worker.config.PollInterval)
	assert.Equal(t, 3, worker.config.MaxRetries)
}

func TestDefaultOutboxWorkerConfig(t *testing.T) {
	config := DefaultOutboxWorkerConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 5, config.WorkerCount)
	assert.Equal(t, 20, config.BatchSize)
	assert.Equal(t, 10*time.Second, config.PollInterval)
	assert.Equal(t, 6, config.MaxRetries)
}

func TestOutboxWorker_CalculateNextRetry(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil, nil)

	testCases := []struct {
		retryCount    int
		expectedDelay time.Duration
		description   string
	}{
		{1, 1 * time.Minute, "Retry 1: 1 minute"},
		{2, 5 * time.Minute, "Retry 2: 5 minutes"},
		{3, 15 * time.Minute, "Retry 3: 15 minutes"},
		{4, 1 * time.Hour, "Retry 4: 1 hour"},
		{5, 4 * time.Hour, "Retry 5: 4 hours"},
		{6, 12 * time.Hour, "Retry 6: 12 hours"},
		{7, 1 * time.Hour, "Retry 7+: default 1 hour"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			nextRetry := worker.calculateNextRetry(tc.retryCount)
			actualDelay := time.Until(nextRetry)

			// Allow for a small margin of error (1 second)
			assert.InDelta(t, tc.expectedDelay.Seconds(), actualDelay.Seconds(), 1.0,
				"Expected delay of %s, got %s", tc.expectedDelay, actualDelay)
		})
	}
}

func TestOutboxWorker_QueueSize(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil, nil)

	// Initially queue should be empty
	assert.Equal(t, 0, worker.QueueSize())
}

func TestOutboxWorker_BuildContentItem_RoundTripsTitleAndAuthor(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil, nil)

	feedID := uuid.New()
	entry := &models.ContentOutbox{
		ID:         uuid.New(),
		FeedItemID: uuid.New(),
		ContentPayload: map[string]interface{}{
			"source_url":     "https://example.com/article",
			"raw_html":       "<html><body><p>body</p></body></html>",
			"source_feed_id": feedID.String(),
			"title":          "RSS-supplied title",
			"author":         "Jane Doe",
		},
	}

	item, err := worker.buildContentItem(entry)
	require.NoError(t, err)

	require.NotNil(t, item.Title)
	assert.Equal(t, "RSS-supplied title", *item.Title)
	require.NotNil(t, item.Author)
	assert.Equal(t, "Jane Doe", *item.Author)
}

func TestOutboxWorker_BuildContentItem_OmitsMissingTitleAndAuthor(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil, nil)

	entry := &models.ContentOutbox{
		ID: uuid.New(),
		ContentPayload: map[string]interface{}{
			"source_url": "https://example.com/article",
			"raw_html":   "<html><body><p>body</p></body></html>",
		},
	}

	item, err := worker.buildContentItem(entry)
	require.NoError(t, err)

	assert.Nil(t, item.Title)
	assert.Nil(t, item.Author)
}

func TestOutboxWorker_BuildContentItem_ErrorsWhenHTMLMissing(t *testing.T) {
	worker := NewOutboxWorker(nil, nil, nil, nil)

	entry := &models.ContentOutbox{
		ID: uuid.New(),
		ContentPayload: map[string]interface{}{
			"source_url": "https://example.com/article",
		},
	}

	_, err := worker.buildContentItem(entry)
	require.Error(t, err)
}

// lockedBuffer is a bytes.Buffer safe for concurrent slog writes and test reads.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureSlog redirects the default slog logger to a buffer for the test.
func captureSlog(t *testing.T) *lockedBuffer {
	t.Helper()
	buf := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// stubOutboxRepo implements only the OutboxRepository methods the worker
// loops call; any other method panics via the nil embedded interface.
type stubOutboxRepo struct {
	repository.OutboxRepository
	getPending     func() ([]*models.ContentOutbox, error)
	incrementRetry func(id uuid.UUID)
}

func (r *stubOutboxRepo) GetPendingEntries(context.Context, int) ([]*models.ContentOutbox, error) {
	return r.getPending()
}

func (r *stubOutboxRepo) IncrementRetryCount(_ context.Context, id uuid.UUID, _ time.Time, _ string) error {
	r.incrementRetry(id)
	return nil
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestOutboxWorker_PollLoop_IdleTickLogsHeartbeat(t *testing.T) {
	buf := captureSlog(t)
	ticks := make(chan struct{}, 16)
	repo := &stubOutboxRepo{getPending: func() ([]*models.ContentOutbox, error) {
		ticks <- struct{}{}
		return nil, nil
	}}
	ow := NewOutboxWorker(&OutboxWorkerConfig{WorkerCount: 1, BatchSize: 1, PollInterval: time.Millisecond}, repo, nil, nil)

	go ow.pollPendingEntries()
	waitFor(t, ticks, "first poll")
	waitFor(t, ticks, "second poll")
	close(ow.stopCh)
	<-ow.pollTickerDone

	assert.Contains(t, buf.String(), `msg="Outbox poll tick" fetched=0 queued=0`)
}

func TestOutboxWorker_PollLoop_RecoversFromPanicAndContinues(t *testing.T) {
	buf := captureSlog(t)
	calls := 0
	second := make(chan struct{})
	repo := &stubOutboxRepo{getPending: func() ([]*models.ContentOutbox, error) {
		calls++
		if calls == 1 {
			panic("boom in poll")
		}
		if calls == 2 {
			close(second)
		}
		return nil, nil
	}}
	ow := NewOutboxWorker(&OutboxWorkerConfig{WorkerCount: 1, BatchSize: 1, PollInterval: time.Millisecond}, repo, nil, nil)

	go ow.pollPendingEntries()
	waitFor(t, second, "poll after panic")
	close(ow.stopCh)
	<-ow.pollTickerDone

	out := buf.String()
	assert.Contains(t, out, "level=ERROR")
	assert.Contains(t, out, "Panic in outbox poll tick")
	assert.Contains(t, out, "boom in poll")
	assert.Contains(t, out, "stack=")
}

func TestOutboxWorker_Worker_RecoversFromPanickingEntryAndProcessesNext(t *testing.T) {
	buf := captureSlog(t)
	bad := &models.ContentOutbox{ID: uuid.New()}
	good := &models.ContentOutbox{ID: uuid.New()}
	handled := make(chan struct{})
	// Entries have no payload, so processing fails to build the content item
	// and reaches IncrementRetryCount, where the first entry panics.
	repo := &stubOutboxRepo{incrementRetry: func(id uuid.UUID) {
		if id == bad.ID {
			panic("boom in entry")
		}
		close(handled)
	}}
	ow := NewOutboxWorker(&OutboxWorkerConfig{WorkerCount: 1, BatchSize: 2, MaxRetries: 6}, repo, nil, nil)

	ow.wg.Add(1)
	go ow.worker(0)
	ow.outboxQueue <- bad
	ow.outboxQueue <- good
	waitFor(t, handled, "second entry after panic")
	close(ow.stopCh)
	ow.wg.Wait()

	out := buf.String()
	assert.Contains(t, out, "Panic processing outbox entry")
	assert.Contains(t, out, "boom in entry")
	assert.True(t, strings.Contains(out, "entry_id="+bad.ID.String()), "panic log should carry the entry ID")
}
