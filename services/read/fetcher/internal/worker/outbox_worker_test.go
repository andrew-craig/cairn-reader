package worker

import (
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/models"
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
