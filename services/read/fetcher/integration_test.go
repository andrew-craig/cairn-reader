//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sharedmw "github.com/cairn-app/cairn-reader/pkg/middleware"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/api/handlers"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/client"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/service"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/testutil"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

// setupTestRouter creates a test router for RSS Fetcher Service
func setupTestRouter(subscriptionHandler *handlers.SubscriptionHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(sharedmw.Recovery)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/users/{user_id}/feeds", func(r chi.Router) {
			r.Post("/subscribe", subscriptionHandler.Subscribe)
			r.Get("/", subscriptionHandler.ListSubscriptions)
			r.Delete("/{feed_id}", subscriptionHandler.Unsubscribe)
		})

		r.Route("/feeds/{feed_id}", func(r chi.Router) {
			r.Patch("/enable", subscriptionHandler.EnableFeed)
		})
	})

	return r
}

// TestContentServiceReceivesRSSTitleAndAuthor verifies that an RSS-parsed
// title and author end up in the HTTP request body sent to the Content
// Service. This is the bug regression test for bug_fda7.
func TestContentServiceReceivesRSSTitleAndAuthor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	rssTitle := "Why Cairns Matter"
	rssAuthor := "Ada Lovelace"
	feedID := uuid.New()

	type captured struct {
		Title  *string `json:"title,omitempty"`
		Author *string `json:"author,omitempty"`
	}
	var receivedItems []captured

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "/api/v1/content/bulk", r.URL.Path)

		var body struct {
			Contents []captured `json:"contents"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		receivedItems = append(receivedItems, body.Contents...)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"created":  []interface{}{map[string]interface{}{"id": uuid.New().String()}},
				"existing": []interface{}{},
				"failed":   []interface{}{},
			},
			"meta": map[string]string{"version": "v1"},
		})
	}))
	defer server.Close()

	contentClient := client.NewContentServiceClient(client.ContentServiceConfig{
		BaseURL: server.URL,
	})

	// Build the BulkContentItem the same way the outbox worker does
	// (services/read/fetcher/internal/worker/outbox_worker.go buildContentItem)
	// so we cover the same JSON contract exposed to the Content Service.
	item := client.BulkContentItem{
		URL:          "https://example.com/cairns",
		HTML:         "<p>body</p>",
		SourceType:   "rss",
		SourceFeedID: &feedID,
		Title:        &rssTitle,
		Author:       &rssAuthor,
	}

	_, err := contentClient.BulkCreateContent(context.Background(), []client.BulkContentItem{item})
	require.NoError(t, err)

	require.Len(t, receivedItems, 1)
	require.NotNil(t, receivedItems[0].Title, "Content Service did not receive title")
	assert.Equal(t, rssTitle, *receivedItems[0].Title)
	require.NotNil(t, receivedItems[0].Author, "Content Service did not receive author")
	assert.Equal(t, rssAuthor, *receivedItems[0].Author)
}

// TestRawHTMLPipelineIntegration is the regression test for the
// double-readability bug: <head>/<meta>/class attributes must reach the
// Content Service untouched so it can run the single readability+sanitize
// pass on the original markup.
func TestRawHTMLPipelineIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	const rawHTML = `<html>
  <head>
    <title>Page Title</title>
    <meta name="author" content="Meta Author">
  </head>
  <body><article class="post"><p>Body</p></article></body>
</html>`

	rssTitle := "RSS Title"
	rssAuthor := "RSS Author"
	feedID := uuid.New()

	type captured struct {
		URL    string  `json:"url"`
		HTML   string  `json:"html"`
		Title  *string `json:"title,omitempty"`
		Author *string `json:"author,omitempty"`
	}
	var got []captured

	csServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "/api/v1/content/bulk", r.URL.Path)

		var body struct {
			Contents []captured `json:"contents"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		got = append(got, body.Contents...)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"created": []interface{}{map[string]interface{}{
					"id":           uuid.New().String(),
					"content_hash": "deadbeef",
				}},
				"existing": []interface{}{},
				"failed":   []interface{}{},
			},
			"meta": map[string]string{"version": "v1"},
		})
	}))
	defer csServer.Close()

	contentClient := client.NewContentServiceClient(client.ContentServiceConfig{BaseURL: csServer.URL})

	item := client.BulkContentItem{
		URL:          "https://example.com/article",
		HTML:         rawHTML,
		SourceType:   "rss",
		SourceFeedID: &feedID,
		Title:        &rssTitle,
		Author:       &rssAuthor,
	}

	_, err := contentClient.BulkCreateContent(context.Background(), []client.BulkContentItem{item})
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, rawHTML, got[0].HTML)
	// <meta> and class attributes are stripped by bluemonday's UGCPolicy. If
	// they survive end-to-end, the fetcher is no longer sanitizing.
	assert.Contains(t, got[0].HTML, `<meta name="author"`)
	assert.Contains(t, got[0].HTML, `class="post"`)
	require.NotNil(t, got[0].Title)
	assert.Equal(t, rssTitle, *got[0].Title)
	require.NotNil(t, got[0].Author)
	assert.Equal(t, rssAuthor, *got[0].Author)
}

// TestFeedSubscriptionIntegration tests the feed subscription flow
func TestFeedSubscriptionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	feedRepo := repository.NewFeedRepository(testDB.DB)
	subscriptionRepo := repository.NewFeedSubscriptionRepository(testDB.DB)
	feedService := service.NewFeedService(feedRepo, subscriptionRepo)
	subscriptionHandler := handlers.NewSubscriptionHandler(feedService)

	router := setupTestRouter(subscriptionHandler)
	server := httptest.NewServer(router)
	defer server.Close()

	userID := uuid.New()

	t.Run("SubscribeToNewFeed", func(t *testing.T) {
		ctx := context.Background()

		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       strPtr("Test Feed"),
			Description: strPtr("A test feed"),
			SiteURL:     strPtr("https://example.com"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		subscription := &models.FeedSubscription{
			UserID: userID,
			FeedID: feed.ID,
		}
		err = subscriptionRepo.Create(ctx, subscription)
		require.NoError(t, err)

		subs, err := subscriptionRepo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, subs, 1)
		assert.Equal(t, feed.ID, subs[0].FeedID)
	})

	t.Run("ListUserFeeds", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/api/v1/users/%s/feeds", server.URL, userID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Content Service / Read Fetcher both wrap successful responses in a
		// {"data": ..., "meta": ...} envelope (see pkg/api.WriteSuccess).
		var envelope struct {
			Data struct {
				Subscriptions []struct {
					FeedID    uuid.UUID `json:"feed_id"`
					FeedURL   string    `json:"feed_url"`
					FeedTitle string    `json:"feed_title"`
				} `json:"subscriptions"`
			} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&envelope)
		require.NoError(t, err)
		assert.Len(t, envelope.Data.Subscriptions, 1)
		assert.Equal(t, "Test Feed", envelope.Data.Subscriptions[0].FeedTitle)
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		ctx := context.Background()
		subs, err := subscriptionRepo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, subs, 1)
		feedID := subs[0].FeedID

		req, err := http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/api/v1/users/%s/feeds/%s", server.URL, userID, feedID),
			nil,
		)
		require.NoError(t, err)

		httpClient := &http.Client{}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		subs, err = subscriptionRepo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, subs, 0)
	})

	t.Run("FeedLimitEnforcement", func(t *testing.T) {
		testDB.TruncateAll()
		ctx := context.Background()

		// Create 100 feeds and subscriptions (the limit).
		for i := 0; i < 100; i++ {
			feed := &models.Feed{
				FeedURL:     fmt.Sprintf("https://example.com/feed%d.xml", i),
				Title:       strPtr(fmt.Sprintf("Feed %d", i)),
				Status:      models.FeedStatusActive,
				PollingTier: models.PollingTierModerate,
			}
			err := feedRepo.Create(ctx, feed)
			require.NoError(t, err)

			subscription := &models.FeedSubscription{
				UserID: userID,
				FeedID: feed.ID,
			}
			err = subscriptionRepo.Create(ctx, subscription)
			require.NoError(t, err)
		}

		count, err := subscriptionRepo.CountByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 100, count)

		// Try to add the 101st feed (the schema-level trigger should reject it).
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed101.xml",
			Title:       strPtr("Feed 101"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
		}
		err = feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		subscription := &models.FeedSubscription{
			UserID: userID,
			FeedID: feed.ID,
		}
		err = subscriptionRepo.Create(ctx, subscription)
		assert.Error(t, err, "Should fail due to trigger constraint")
		assert.Contains(t, err.Error(), "user has reached the maximum of 100 feed subscriptions")
	})
}

// TestOutboxPatternIntegration tests the outbox pattern delivery flow
func TestOutboxPatternIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	ctx := context.Background()
	outboxRepo := repository.NewOutboxRepository(testDB.DB)

	t.Run("CreateOutboxEntry", func(t *testing.T) {
		userIDs := []uuid.UUID{uuid.New(), uuid.New()}

		contentPayload := map[string]interface{}{
			"url":          "https://example.com/article",
			"title":        "Test Article",
			"cleaned_html": "<p>Test</p>",
			"source_type":  "rss",
		}

		outboxEntry := &models.ContentOutbox{
			FeedItemID:     uuid.New(),
			UserIDs:        userIDs,
			ContentPayload: contentPayload,
			DeliveryStatus: models.DeliveryStatusPending,
			RetryCount:     0,
		}

		err := outboxRepo.Create(ctx, outboxEntry)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, outboxEntry.ID)

		entry, err := outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DeliveryStatusPending, entry.DeliveryStatus)
		assert.Len(t, entry.UserIDs, 2)
	})

	t.Run("GetPendingEntries", func(t *testing.T) {
		testDB.TruncateAll()

		for i := 0; i < 5; i++ {
			outboxEntry := &models.ContentOutbox{
				FeedItemID: uuid.New(),
				UserIDs:    []uuid.UUID{uuid.New()},
				ContentPayload: map[string]interface{}{
					"title": fmt.Sprintf("Article %d", i),
				},
				DeliveryStatus: models.DeliveryStatusPending,
				NextRetryAt:    time.Now().Add(-1 * time.Hour),
			}
			err := outboxRepo.Create(ctx, outboxEntry)
			require.NoError(t, err)
		}

		pending, err := outboxRepo.GetPendingEntries(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, pending, 5)
	})

	t.Run("UpdateDeliveryStatus", func(t *testing.T) {
		testDB.TruncateAll()

		outboxEntry := &models.ContentOutbox{
			FeedItemID:     uuid.New(),
			UserIDs:        []uuid.UUID{uuid.New()},
			ContentPayload: map[string]interface{}{"title": "Test"},
			DeliveryStatus: models.DeliveryStatusPending,
		}
		err := outboxRepo.Create(ctx, outboxEntry)
		require.NoError(t, err)

		err = outboxRepo.UpdateDeliveryStatus(ctx, outboxEntry.ID, models.DeliveryStatusSending, nil, nil, nil)
		require.NoError(t, err)

		entry, err := outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DeliveryStatusSending, entry.DeliveryStatus)

		contentServiceID := uuid.New()
		now := time.Now()
		err = outboxRepo.UpdateDeliveryStatus(ctx, outboxEntry.ID, models.DeliveryStatusDelivered, &contentServiceID, &now, nil)
		require.NoError(t, err)

		entry, err = outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DeliveryStatusDelivered, entry.DeliveryStatus)
		assert.NotNil(t, entry.DeliveredAt)
		require.NotNil(t, entry.ContentServiceID)
		assert.Equal(t, contentServiceID, *entry.ContentServiceID)
	})

	t.Run("RetryLogic", func(t *testing.T) {
		testDB.TruncateAll()

		outboxEntry := &models.ContentOutbox{
			FeedItemID:     uuid.New(),
			UserIDs:        []uuid.UUID{uuid.New()},
			ContentPayload: map[string]interface{}{"title": "Test"},
			DeliveryStatus: models.DeliveryStatusPending,
			RetryCount:     0,
		}
		err := outboxRepo.Create(ctx, outboxEntry)
		require.NoError(t, err)

		nextRetry := time.Now().Add(5 * time.Minute)
		err = outboxRepo.IncrementRetryCount(ctx, outboxEntry.ID, nextRetry, "Connection timeout")
		require.NoError(t, err)

		entry, err := outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, entry.RetryCount)
		assert.Equal(t, models.DeliveryStatusPending, entry.DeliveryStatus)
		assert.NotNil(t, entry.LastError)
	})
}

// TestFeedItemProcessingIntegration tests RSS item processing and deduplication
func TestFeedItemProcessingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	ctx := context.Background()
	feedRepo := repository.NewFeedRepository(testDB.DB)
	feedItemRepo := repository.NewFeedItemRepository(testDB.DB)

	feed := &models.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       strPtr("Test Feed"),
		Status:      models.FeedStatusActive,
		PollingTier: models.PollingTierModerate,
	}
	err := feedRepo.Create(ctx, feed)
	require.NoError(t, err)

	t.Run("CreateFeedItem", func(t *testing.T) {
		publishedAt := time.Now().Add(-1 * time.Hour)

		feedItem := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         strPtr("unique-guid-123"),
			Title:            strPtr("Test Article"),
			Author:           strPtr("John Doe"),
			PublishedAt:      &publishedAt,
			Description:      strPtr("Article description"),
			ItemURL:          "https://example.com/article1",
			ProcessingStatus: models.ProcessingStatusPending,
		}

		err := feedItemRepo.Create(ctx, feedItem)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, feedItem.ID)

		item, err := feedItemRepo.GetByID(ctx, feedItem.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessingStatusPending, item.ProcessingStatus)
		require.NotNil(t, item.Title)
		assert.Equal(t, "Test Article", *item.Title)
	})

	t.Run("DeduplicationByGUID", func(t *testing.T) {
		testDB.TruncateAll()

		// Recreate feed after truncate.
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       strPtr("Test Feed"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		publishedAt := time.Now()

		feedItem1 := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         strPtr("duplicate-guid"),
			Title:            strPtr("Article 1"),
			ItemURL:          "https://example.com/article",
			ProcessingStatus: models.ProcessingStatusPending,
			PublishedAt:      &publishedAt,
		}
		err = feedItemRepo.Create(ctx, feedItem1)
		require.NoError(t, err)

		feedItem2 := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         strPtr("duplicate-guid"),
			Title:            strPtr("Article 2 (Duplicate)"),
			ItemURL:          "https://example.com/article",
			ProcessingStatus: models.ProcessingStatusPending,
			PublishedAt:      &publishedAt,
		}
		err = feedItemRepo.Create(ctx, feedItem2)
		assert.Error(t, err, "Should fail due to unique constraint")
		assert.Contains(t, err.Error(), "duplicate key value")
	})

	t.Run("GetPendingItems", func(t *testing.T) {
		testDB.TruncateAll()

		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       strPtr("Test Feed"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		publishedAt := time.Now()
		for i := 0; i < 3; i++ {
			feedItem := &models.FeedItem{
				FeedID:           feed.ID,
				ItemGUID:         strPtr(fmt.Sprintf("guid-%d", i)),
				Title:            strPtr(fmt.Sprintf("Article %d", i)),
				ItemURL:          fmt.Sprintf("https://example.com/article%d", i),
				ProcessingStatus: models.ProcessingStatusPending,
				PublishedAt:      &publishedAt,
			}
			err := feedItemRepo.Create(ctx, feedItem)
			require.NoError(t, err)
		}

		completedItem := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         strPtr("completed-guid"),
			Title:            strPtr("Completed Article"),
			ItemURL:          "https://example.com/completed",
			ProcessingStatus: models.ProcessingStatusCompleted,
			PublishedAt:      &publishedAt,
		}
		err = feedItemRepo.Create(ctx, completedItem)
		require.NoError(t, err)

		pending, err := feedItemRepo.GetPendingItems(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, pending, 3, "Should only return pending items")
	})

	t.Run("UpdateProcessingStatus", func(t *testing.T) {
		testDB.TruncateAll()

		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       strPtr("Test Feed"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		publishedAt := time.Now()
		feedItem := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         strPtr("test-guid"),
			Title:            strPtr("Test Article"),
			ItemURL:          "https://example.com/test",
			ProcessingStatus: models.ProcessingStatusPending,
			PublishedAt:      &publishedAt,
		}
		err = feedItemRepo.Create(ctx, feedItem)
		require.NoError(t, err)

		// Move to processing and stamp the content hash.
		contentHash := "abc123hash"
		err = feedItemRepo.UpdateProcessingStatus(ctx, feedItem.ID, models.ProcessingStatusProcessing, &contentHash, nil, nil)
		require.NoError(t, err)

		item, err := feedItemRepo.GetByID(ctx, feedItem.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessingStatusProcessing, item.ProcessingStatus)
		require.NotNil(t, item.ContentHash)
		assert.Equal(t, contentHash, *item.ContentHash)

		// Mark as completed.
		now := time.Now()
		err = feedItemRepo.UpdateProcessingStatus(ctx, feedItem.ID, models.ProcessingStatusCompleted, nil, nil, &now)
		require.NoError(t, err)

		item, err = feedItemRepo.GetByID(ctx, feedItem.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessingStatusCompleted, item.ProcessingStatus)
	})
}

// TestFeedPollingIntegration tests feed polling logic
func TestFeedPollingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Cleanup()

	ctx := context.Background()
	feedRepo := repository.NewFeedRepository(testDB.DB)

	t.Run("GetFeedsDueForPolling", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		futureTime := time.Now().Add(1 * time.Hour)

		feed1 := &models.Feed{
			FeedURL:     "https://example.com/feed1.xml",
			Title:       strPtr("Due Feed 1"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierActive,
			NextPollAt:  pastTime,
		}
		err := feedRepo.Create(ctx, feed1)
		require.NoError(t, err)

		feed2 := &models.Feed{
			FeedURL:     "https://example.com/feed2.xml",
			Title:       strPtr("Due Feed 2"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
			NextPollAt:  pastTime,
		}
		err = feedRepo.Create(ctx, feed2)
		require.NoError(t, err)

		feed3 := &models.Feed{
			FeedURL:     "https://example.com/feed3.xml",
			Title:       strPtr("Not Due Feed"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierQuiet,
			NextPollAt:  futureTime,
		}
		err = feedRepo.Create(ctx, feed3)
		require.NoError(t, err)

		dueFeeds, err := feedRepo.GetFeedsDueForPolling(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, dueFeeds, 2, "Should return 2 feeds due for polling")
	})

	t.Run("UpdateFeedAfterPoll", func(t *testing.T) {
		testDB.TruncateAll()

		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       strPtr("Test Feed"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
			NextPollAt:  time.Now().Add(-1 * time.Hour),
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		now := time.Now()
		nextPollTime := now.Add(6 * time.Hour) // Moderate tier
		err = feedRepo.UpdatePollingInfo(ctx, feed.ID, &now, nil, nextPollTime, models.PollingTierModerate)
		require.NoError(t, err)

		// Reset error tracking on successful poll.
		err = feedRepo.UpdateErrorInfo(ctx, feed.ID, 0, nil, nil)
		require.NoError(t, err)

		updated, err := feedRepo.GetByID(ctx, feed.ID)
		require.NoError(t, err)
		assert.NotNil(t, updated.LastFetchedAt)
		assert.Equal(t, 0, updated.ConsecutiveErrorDays)
	})

	t.Run("TrackConsecutiveErrors", func(t *testing.T) {
		testDB.TruncateAll()

		feed := &models.Feed{
			FeedURL:     "https://example.com/failing-feed.xml",
			Title:       strPtr("Failing Feed"),
			Status:      models.FeedStatusActive,
			PollingTier: models.PollingTierModerate,
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		// Simulate 7 consecutive days of errors and assert the counter advances.
		// Auto-disable behaviour (when present) lives in the service layer; this
		// test only covers the repository-level error tracking.
		for i := 1; i <= 7; i++ {
			now := time.Now()
			msg := fmt.Sprintf("Error day %d", i)
			err = feedRepo.UpdateErrorInfo(ctx, feed.ID, i, &now, &msg)
			require.NoError(t, err)

			updated, err := feedRepo.GetByID(ctx, feed.ID)
			require.NoError(t, err)
			assert.Equal(t, i, updated.ConsecutiveErrorDays)
		}
	})
}
