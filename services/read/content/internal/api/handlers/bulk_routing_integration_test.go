//go:build integration
// +build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestBulkIngest_RoutesByUserAndSource_Integration drives the real ingest path
// (POST bulk-create, then POST internal bulk-add) against Postgres and checks
// that each user's item lands in the list their route for that source says.
func TestBulkIngest_RoutesByUserAndSource_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	contentRepo := repository.NewContentRepository(testDB.DB)
	ucRepo := repository.NewUserContentRepository(testDB.DB)
	routeRepo := repository.NewSourceRouteRepository(testDB.DB)
	handler := NewBulkHandler(service.NewContentService(contentRepo, testDB.DB), ucRepo, contentRepo)
	ctx := context.Background()

	feedUser, readsUser, unroutedUser := uuid.New(), uuid.New(), uuid.New()
	feedID, senderID := uuid.New(), uuid.New()

	for _, r := range []*models.SourceRoute{
		{UserID: feedUser, SourceType: models.SourceTypeRSS, SourceKey: feedID, List: models.ListFeed},
		{UserID: readsUser, SourceType: models.SourceTypeRSS, SourceKey: feedID, List: models.ListReads},
		{UserID: feedUser, SourceType: models.SourceTypeEmail, SourceKey: senderID, List: models.ListFeed},
	} {
		require.NoError(t, routeRepo.Upsert(ctx, r))
	}

	post := func(call http.HandlerFunc, body any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		call(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		return w
	}

	// createContent mirrors what the fetcher / email worker send.
	createContent := func(item map[string]any) uuid.UUID {
		w := post(handler.BulkCreateContent, map[string]any{"contents": []any{item}})
		var resp struct {
			Data struct {
				Created []struct {
					ID uuid.UUID `json:"id"`
				} `json:"created"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
		require.Len(t, resp.Data.Created, 1)
		return resp.Data.Created[0].ID
	}
	deliver := func(contentID uuid.UUID, users ...uuid.UUID) {
		items := make([]map[string]any, len(users))
		for i, u := range users {
			items[i] = map[string]any{"content_id": contentID, "user_id": u, "status": "unread"}
		}
		post(handler.BulkAddToUsersInternal, map[string]any{"items": items})
	}
	listOf := func(userID, contentID uuid.UUID) string {
		uc, err := ucRepo.GetByUserAndContent(ctx, userID, contentID)
		require.NoError(t, err)
		require.NotNil(t, uc, "user %s should have the item", userID)
		return uc.List
	}

	// One RSS item delivered to three users in a single request, as the fetcher outbox does.
	rssID := createContent(map[string]any{
		"url": "https://example.com/rss-item", "html": "<html><body><p>rss body</p></body></html>",
		"source_type": "rss", "source_feed_id": feedID,
	})
	deliver(rssID, feedUser, readsUser, unroutedUser)
	require.Equal(t, models.ListFeed, listOf(feedUser, rssID), "RSS -> feed")
	require.Equal(t, models.ListReads, listOf(readsUser, rssID), "RSS -> reads")
	require.Equal(t, models.ListReads, listOf(unroutedUser, rssID), "RSS, no route -> reads")

	// Email: the sender ID travels on the content row and the route is looked up through it.
	emailID := createContent(map[string]any{
		"url": "email://" + uuid.NewString(), "html": "<html><body><p>newsletter</p></body></html>",
		"source_type": "email", "source_sender_id": senderID,
	})
	stored, err := contentRepo.GetByID(ctx, emailID)
	require.NoError(t, err)
	require.NotNil(t, stored.SourceSenderID)
	require.Equal(t, senderID, *stored.SourceSenderID, "content service must store the sender ID")

	deliver(emailID, feedUser, unroutedUser)
	require.Equal(t, models.ListFeed, listOf(feedUser, emailID), "email sender -> feed")
	require.Equal(t, models.ListReads, listOf(unroutedUser, emailID), "email sender, no route -> reads")

	// Re-routing affects only items delivered afterwards.
	require.NoError(t, routeRepo.Upsert(ctx, &models.SourceRoute{
		UserID: feedUser, SourceType: models.SourceTypeRSS, SourceKey: feedID, List: models.ListReads}))
	laterID := createContent(map[string]any{
		"url": "https://example.com/rss-item-2", "html": "<html><body><p>second</p></body></html>",
		"source_type": "rss", "source_feed_id": feedID,
	})
	deliver(laterID, feedUser)
	require.Equal(t, models.ListReads, listOf(feedUser, laterID), "new item follows the new route")
	require.Equal(t, models.ListFeed, listOf(feedUser, rssID), "existing item is not moved")
}
