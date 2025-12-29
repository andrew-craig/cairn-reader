// +build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-read/fetcher/internal/api/handlers"
	"github.com/andrew-craig/cairn-read/fetcher/internal/api/middleware"
	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-read/fetcher/internal/repository"
	"github.com/andrew-craig/cairn-read/fetcher/internal/service"
	"github.com/andrew-craig/cairn-read/fetcher/internal/testhelpers"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a test router for RSS Fetcher Service
func setupTestRouter(subscriptionHandler *handlers.SubscriptionHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
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

// TestFeedSubscriptionIntegration tests the feed subscription flow
func TestFeedSubscriptionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testhelpers.SetupTestDatabase(t)
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
		// Note: This test will fail without a real RSS feed
		// For integration testing, we'll insert a feed directly
		ctx := context.Background()

		// Create a feed directly in the database
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       "Test Feed",
			Description: "A test feed",
			SiteURL:     "https://example.com",
			Status:      "active",
			PollingTier: "moderate",
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		// Create subscription
		subscription := &models.FeedSubscription{
			UserID: userID,
			FeedID: feed.ID,
		}
		err = subscriptionRepo.Create(ctx, subscription)
		require.NoError(t, err)

		// Verify subscription was created
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

		var response struct {
			Subscriptions []struct {
				FeedID      uuid.UUID `json:"feed_id"`
				FeedURL     string    `json:"feed_url"`
				Title       string    `json:"title"`
				Description string    `json:"description"`
			} `json:"subscriptions"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response.Subscriptions, 1)
		assert.Equal(t, "Test Feed", response.Subscriptions[0].Title)
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		// Get feed ID first
		ctx := context.Background()
		subs, err := subscriptionRepo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		require.Len(t, subs, 1)
		feedID := subs[0].FeedID

		// Unsubscribe
		req, err := http.NewRequest(
			"DELETE",
			fmt.Sprintf("%s/api/v1/users/%s/feeds/%s", server.URL, userID, feedID),
			nil,
		)
		require.NoError(t, err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// Verify unsubscription
		subs, err = subscriptionRepo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, subs, 0)
	})

	t.Run("FeedLimitEnforcement", func(t *testing.T) {
		testDB.TruncateAll()
		ctx := context.Background()

		// Create 100 feeds and subscriptions (the limit)
		for i := 0; i < 100; i++ {
			feed := &models.Feed{
				FeedURL:     fmt.Sprintf("https://example.com/feed%d.xml", i),
				Title:       fmt.Sprintf("Feed %d", i),
				Status:      "active",
				PollingTier: "moderate",
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

		// Verify count
		count, err := subscriptionRepo.CountByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 100, count)

		// Try to add 101st feed (should fail)
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed101.xml",
			Title:       "Feed 101",
			Status:      "active",
			PollingTier: "moderate",
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

	testDB := testhelpers.SetupTestDatabase(t)
	defer testDB.Cleanup()

	ctx := context.Background()
	outboxRepo := repository.NewOutboxRepository(testDB.DB)

	t.Run("CreateOutboxEntry", func(t *testing.T) {
		userIDs := []uuid.UUID{uuid.New(), uuid.New()}

		contentPayload := map[string]interface{}{
			"url":         "https://example.com/article",
			"title":       "Test Article",
			"cleaned_html": "<p>Test</p>",
			"source_type": "rss",
		}

		outboxEntry := &models.ContentOutbox{
			FeedItemID:     uuid.New(),
			UserIDs:        userIDs,
			ContentPayload: contentPayload,
			DeliveryStatus: "pending",
			RetryCount:     0,
		}

		err := outboxRepo.Create(ctx, outboxEntry)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, outboxEntry.ID)

		// Verify entry was created
		entry, err := outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, "pending", entry.DeliveryStatus)
		assert.Len(t, entry.UserIDs, 2)
	})

	t.Run("GetPendingEntries", func(t *testing.T) {
		testDB.TruncateAll()

		// Create multiple pending entries
		for i := 0; i < 5; i++ {
			outboxEntry := &models.ContentOutbox{
				FeedItemID: uuid.New(),
				UserIDs:    []uuid.UUID{uuid.New()},
				ContentPayload: map[string]interface{}{
					"title": fmt.Sprintf("Article %d", i),
				},
				DeliveryStatus: "pending",
				NextRetryAt:    time.Now().Add(-1 * time.Hour), // Due for delivery
			}
			err := outboxRepo.Create(ctx, outboxEntry)
			require.NoError(t, err)
		}

		// Get pending entries
		pending, err := outboxRepo.GetPending(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, pending, 5)
	})

	t.Run("UpdateDeliveryStatus", func(t *testing.T) {
		testDB.TruncateAll()

		outboxEntry := &models.ContentOutbox{
			FeedItemID:     uuid.New(),
			UserIDs:        []uuid.UUID{uuid.New()},
			ContentPayload: map[string]interface{}{"title": "Test"},
			DeliveryStatus: "pending",
		}
		err := outboxRepo.Create(ctx, outboxEntry)
		require.NoError(t, err)

		// Update to sending
		err = outboxRepo.UpdateStatus(ctx, outboxEntry.ID, "sending")
		require.NoError(t, err)

		// Verify update
		entry, err := outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, "sending", entry.DeliveryStatus)

		// Mark as delivered
		contentServiceID := uuid.New()
		err = outboxRepo.MarkDelivered(ctx, outboxEntry.ID, contentServiceID)
		require.NoError(t, err)

		// Verify delivery
		entry, err = outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, "delivered", entry.DeliveryStatus)
		assert.NotNil(t, entry.DeliveredAt)
		assert.Equal(t, &contentServiceID, entry.ContentServiceID)
	})

	t.Run("RetryLogic", func(t *testing.T) {
		testDB.TruncateAll()

		outboxEntry := &models.ContentOutbox{
			FeedItemID:     uuid.New(),
			UserIDs:        []uuid.UUID{uuid.New()},
			ContentPayload: map[string]interface{}{"title": "Test"},
			DeliveryStatus: "pending",
			RetryCount:     0,
		}
		err := outboxRepo.Create(ctx, outboxEntry.ID)
		require.NoError(t, err)

		// Increment retry count
		nextRetry := time.Now().Add(5 * time.Minute)
		err = outboxRepo.IncrementRetry(ctx, outboxEntry.ID, nextRetry, "Connection timeout")
		require.NoError(t, err)

		// Verify retry state
		entry, err := outboxRepo.GetByID(ctx, outboxEntry.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, entry.RetryCount)
		assert.Equal(t, "pending", entry.DeliveryStatus)
		assert.NotNil(t, entry.LastError)
	})
}

// TestFeedItemProcessingIntegration tests RSS item processing and deduplication
func TestFeedItemProcessingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testhelpers.SetupTestDatabase(t)
	defer testDB.Cleanup()

	ctx := context.Background()
	feedRepo := repository.NewFeedRepository(testDB.DB)
	feedItemRepo := repository.NewFeedItemRepository(testDB.DB)

	// Create a feed
	feed := &models.Feed{
		FeedURL:     "https://example.com/feed.xml",
		Title:       "Test Feed",
		Status:      "active",
		PollingTier: "moderate",
	}
	err := feedRepo.Create(ctx, feed)
	require.NoError(t, err)

	t.Run("CreateFeedItem", func(t *testing.T) {
		publishedAt := time.Now().Add(-1 * time.Hour)

		feedItem := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         "unique-guid-123",
			Title:            "Test Article",
			Author:           "John Doe",
			PublishedAt:      &publishedAt,
			Description:      "Article description",
			URL:              "https://example.com/article1",
			ContentHash:      "",
			ProcessingStatus: "pending",
		}

		err := feedItemRepo.Create(ctx, feedItem)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, feedItem.ID)

		// Verify creation
		item, err := feedItemRepo.GetByID(ctx, feedItem.ID)
		require.NoError(t, err)
		assert.Equal(t, "pending", item.ProcessingStatus)
		assert.Equal(t, "Test Article", item.Title)
	})

	t.Run("DeduplicationByGUID", func(t *testing.T) {
		testDB.TruncateAll()

		// Recreate feed after truncate
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       "Test Feed",
			Status:      "active",
			PollingTier: "moderate",
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		publishedAt := time.Now()

		feedItem1 := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         "duplicate-guid",
			Title:            "Article 1",
			URL:              "https://example.com/article",
			ProcessingStatus: "pending",
			PublishedAt:      &publishedAt,
		}

		err = feedItemRepo.Create(ctx, feedItem1)
		require.NoError(t, err)

		// Try to create duplicate (same feed_id + item_guid)
		feedItem2 := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         "duplicate-guid",
			Title:            "Article 2 (Duplicate)",
			URL:              "https://example.com/article",
			ProcessingStatus: "pending",
			PublishedAt:      &publishedAt,
		}

		err = feedItemRepo.Create(ctx, feedItem2)
		assert.Error(t, err, "Should fail due to unique constraint")
		assert.Contains(t, err.Error(), "duplicate key value")
	})

	t.Run("GetPendingItems", func(t *testing.T) {
		testDB.TruncateAll()

		// Recreate feed
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       "Test Feed",
			Status:      "active",
			PollingTier: "moderate",
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		// Create multiple items with different statuses
		publishedAt := time.Now()
		for i := 0; i < 3; i++ {
			feedItem := &models.FeedItem{
				FeedID:           feed.ID,
				ItemGUID:         fmt.Sprintf("guid-%d", i),
				Title:            fmt.Sprintf("Article %d", i),
				URL:              fmt.Sprintf("https://example.com/article%d", i),
				ProcessingStatus: "pending",
				PublishedAt:      &publishedAt,
			}
			err := feedItemRepo.Create(ctx, feedItem)
			require.NoError(t, err)
		}

		// Create one completed item
		completedItem := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         "completed-guid",
			Title:            "Completed Article",
			URL:              "https://example.com/completed",
			ProcessingStatus: "completed",
			PublishedAt:      &publishedAt,
		}
		err = feedItemRepo.Create(ctx, completedItem)
		require.NoError(t, err)

		// Get pending items
		pending, err := feedItemRepo.GetPending(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, pending, 3, "Should only return pending items")
	})

	t.Run("UpdateProcessingStatus", func(t *testing.T) {
		testDB.TruncateAll()

		// Recreate feed
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       "Test Feed",
			Status:      "active",
			PollingTier: "moderate",
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		publishedAt := time.Now()
		feedItem := &models.FeedItem{
			FeedID:           feed.ID,
			ItemGUID:         "test-guid",
			Title:            "Test Article",
			URL:              "https://example.com/test",
			ProcessingStatus: "pending",
			PublishedAt:      &publishedAt,
		}
		err = feedItemRepo.Create(ctx, feedItem)
		require.NoError(t, err)

		// Update to processing
		contentHash := "abc123hash"
		err = feedItemRepo.UpdateContentHash(ctx, feedItem.ID, contentHash)
		require.NoError(t, err)

		err = feedItemRepo.UpdateStatus(ctx, feedItem.ID, "processing")
		require.NoError(t, err)

		// Verify update
		item, err := feedItemRepo.GetByID(ctx, feedItem.ID)
		require.NoError(t, err)
		assert.Equal(t, "processing", item.ProcessingStatus)
		assert.Equal(t, contentHash, item.ContentHash)

		// Mark as completed
		err = feedItemRepo.UpdateStatus(ctx, feedItem.ID, "completed")
		require.NoError(t, err)

		item, err = feedItemRepo.GetByID(ctx, feedItem.ID)
		require.NoError(t, err)
		assert.Equal(t, "completed", item.ProcessingStatus)
	})
}

// TestFeedPollingIntegration tests feed polling logic
func TestFeedPollingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testhelpers.SetupTestDatabase(t)
	defer testDB.Cleanup()

	ctx := context.Background()
	feedRepo := repository.NewFeedRepository(testDB.DB)

	t.Run("GetFeedsDueForPolling", func(t *testing.T) {
		// Create feeds with different next_poll_at times
		pastTime := time.Now().Add(-1 * time.Hour)
		futureTime := time.Now().Add(1 * time.Hour)

		feed1 := &models.Feed{
			FeedURL:     "https://example.com/feed1.xml",
			Title:       "Due Feed 1",
			Status:      "active",
			PollingTier: "active",
			NextPollAt:  &pastTime,
		}
		err := feedRepo.Create(ctx, feed1)
		require.NoError(t, err)

		feed2 := &models.Feed{
			FeedURL:     "https://example.com/feed2.xml",
			Title:       "Due Feed 2",
			Status:      "active",
			PollingTier: "moderate",
			NextPollAt:  &pastTime,
		}
		err = feedRepo.Create(ctx, feed2)
		require.NoError(t, err)

		feed3 := &models.Feed{
			FeedURL:     "https://example.com/feed3.xml",
			Title:       "Not Due Feed",
			Status:      "active",
			PollingTier: "quiet",
			NextPollAt:  &futureTime,
		}
		err = feedRepo.Create(ctx, feed3)
		require.NoError(t, err)

		// Get feeds due for polling
		dueFeeds, err := feedRepo.GetFeedsDueForPolling(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, dueFeeds, 2, "Should return 2 feeds due for polling")
	})

	t.Run("UpdateFeedAfterPoll", func(t *testing.T) {
		testDB.TruncateAll()

		nextPoll := time.Now().Add(-1 * time.Hour)
		feed := &models.Feed{
			FeedURL:     "https://example.com/feed.xml",
			Title:       "Test Feed",
			Status:      "active",
			PollingTier: "moderate",
			NextPollAt:  &nextPoll,
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		// Update after successful poll
		now := time.Now()
		nextPollTime := now.Add(6 * time.Hour) // Moderate tier
		err = feedRepo.UpdateAfterSuccessfulPoll(ctx, feed.ID, nextPollTime)
		require.NoError(t, err)

		// Verify update
		updated, err := feedRepo.GetByID(ctx, feed.ID)
		require.NoError(t, err)
		assert.NotNil(t, updated.LastFetchedAt)
		assert.Equal(t, 0, updated.ConsecutiveErrorDays)
	})

	t.Run("TrackConsecutiveErrors", func(t *testing.T) {
		testDB.TruncateAll()

		feed := &models.Feed{
			FeedURL:     "https://example.com/failing-feed.xml",
			Title:       "Failing Feed",
			Status:      "active",
			PollingTier: "moderate",
		}
		err := feedRepo.Create(ctx, feed)
		require.NoError(t, err)

		// Record errors for 7 consecutive days
		for i := 1; i <= 7; i++ {
			err = feedRepo.RecordError(ctx, feed.ID, fmt.Sprintf("Error day %d", i))
			require.NoError(t, err)

			updated, err := feedRepo.GetByID(ctx, feed.ID)
			require.NoError(t, err)
			assert.Equal(t, i, updated.ConsecutiveErrorDays)

			if i >= 7 {
				assert.Equal(t, "disabled", updated.Status, "Should be disabled after 7 errors")
			}
		}
	})
}
