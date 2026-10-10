package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// routedRequest builds a request carrying chi URL params and an authenticated user.
func routedRequest(method, target string, body interface{}, authUserID uuid.UUID, params map[string]string) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return addAuthContextToRequest(req, authUserID)
}

// fetcherSubscribeServer fakes the Ingest RSS subscribe endpoint.
func fetcherSubscribeServer(t *testing.T, feedID uuid.UUID) *service.IngestRSSClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteSuccess(w, http.StatusCreated, map[string]any{
			"subscription_id": uuid.NewString(),
			"feed_id":         feedID.String(),
			"feed_url":        "https://example.com/feed.xml",
			"feed_title":      "Example",
			"subscribed_at":   time.Now(),
		}, "v1")
	}))
	t.Cleanup(srv.Close)
	return service.NewIngestRSSClient(srv.URL, "test-key")
}

// --- Subscribe: the chosen list is written as the feed's route ---

// With no explicit list nothing is written: no route already means Reads, and a
// route the user set earlier must not be overwritten by a bare re-subscribe.
func TestAddContentToUser_FeedSubscription_OmittedListWritesNoRoute(t *testing.T) {
	userID, feedID := uuid.New(), uuid.New()
	routes := new(MockSourceRouteRepository)
	handler := NewUserContentHandler(new(MockUserContentRepository), new(MockContentRepository), routes,
		nil, nil, fetcherSubscribeServer(t, feedID))

	w := httptest.NewRecorder()
	handler.AddContentToUser(w, routedRequest(http.MethodPost, "/",
		map[string]any{"url": "https://example.com/feed.xml", "type": "feed"},
		userID, map[string]string{"user_id": userID.String()}))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	routes.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)

	var resp struct {
		Data struct {
			Subscription struct {
				List string `json:"list"`
			} `json:"subscription"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, models.ListReads, resp.Data.Subscription.List)
}

// list only means something for a feed; pages and content-ID saves always go to Reads.
func TestAddContentToUser_ListRejectedOnNonFeedSubmission(t *testing.T) {
	userID := uuid.New()
	for name, body := range map[string]map[string]any{
		"page":       {"url": "https://example.com/post", "type": "page", "list": "feed"},
		"content id": {"content_id": uuid.NewString(), "list": "reads"},
	} {
		t.Run(name, func(t *testing.T) {
			ucRepo := new(MockUserContentRepository)
			handler := NewUserContentHandler(ucRepo, new(MockContentRepository), new(MockSourceRouteRepository), nil, nil, nil)

			w := httptest.NewRecorder()
			handler.AddContentToUser(w, routedRequest(http.MethodPost, "/", body, userID, map[string]string{"user_id": userID.String()}))

			assert.Equal(t, http.StatusBadRequest, w.Code)
			ucRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestAddContentToUser_FeedSubscription_WritesRoute(t *testing.T) {
	tests := []struct {
		name     string
		reqList  string
		wantList string
	}{
		{"explicit feed", "feed", models.ListFeed},
		{"explicit reads", "reads", models.ListReads},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, feedID := uuid.New(), uuid.New()
			routes := new(MockSourceRouteRepository)
			routes.On("Upsert", mock.Anything, &models.SourceRoute{
				UserID: userID, SourceType: models.SourceTypeRSS, SourceKey: feedID, List: tt.wantList,
			}).Return(nil)

			handler := NewUserContentHandler(new(MockUserContentRepository), new(MockContentRepository), routes,
				nil, nil, fetcherSubscribeServer(t, feedID))

			body := map[string]any{"url": "https://example.com/feed.xml", "type": "feed"}
			if tt.reqList != "" {
				body["list"] = tt.reqList
			}
			w := httptest.NewRecorder()
			handler.AddContentToUser(w, routedRequest(http.MethodPost, "/", body, userID, map[string]string{"user_id": userID.String()}))

			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			routes.AssertExpectations(t)

			var resp struct {
				Data struct {
					Subscription struct {
						List string `json:"list"`
					} `json:"subscription"`
				} `json:"data"`
			}
			require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
			assert.Equal(t, tt.wantList, resp.Data.Subscription.List)
		})
	}
}

func TestAddContentToUser_InvalidList(t *testing.T) {
	userID := uuid.New()
	routes := new(MockSourceRouteRepository)
	handler := NewUserContentHandler(new(MockUserContentRepository), new(MockContentRepository), routes, nil, nil, nil)

	w := httptest.NewRecorder()
	handler.AddContentToUser(w, routedRequest(http.MethodPost, "/",
		map[string]any{"url": "https://example.com/feed.xml", "type": "feed", "list": "explore"},
		userID, map[string]string{"user_id": userID.String()}))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	routes.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
}

func TestAddContentToUser_FeedSubscription_RouteWriteFails(t *testing.T) {
	userID, feedID := uuid.New(), uuid.New()
	routes := new(MockSourceRouteRepository)
	routes.On("Upsert", mock.Anything, mock.Anything).Return(errors.New("db down"))
	handler := NewUserContentHandler(new(MockUserContentRepository), new(MockContentRepository), routes,
		nil, nil, fetcherSubscribeServer(t, feedID))

	w := httptest.NewRecorder()
	handler.AddContentToUser(w, routedRequest(http.MethodPost, "/",
		map[string]any{"url": "https://example.com/feed.xml", "type": "feed", "list": "feed"},
		userID, map[string]string{"user_id": userID.String()}))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Unsubscribe: the route goes with the subscription ---

func TestUnsubscribeRSS_DeletesRoute(t *testing.T) {
	userID, feedID := uuid.New(), uuid.New()
	fetcher := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer fetcher.Close()

	routes := new(MockSourceRouteRepository)
	routes.On("Delete", mock.Anything, userID, models.SourceTypeRSS, feedID).Return(nil)
	handler := NewSubscriptionAggregatorHandler(service.NewIngestRSSClient(fetcher.URL, "k"), nil, routes)

	w := httptest.NewRecorder()
	handler.UnsubscribeRSS(w, routedRequest(http.MethodDelete, "/", nil, userID,
		map[string]string{"user_id": userID.String(), "feed_id": feedID.String()}))

	assert.Equal(t, http.StatusOK, w.Code)
	routes.AssertExpectations(t)
}

func TestUnsubscribeRSS_NotFound_KeepsRoute(t *testing.T) {
	userID, feedID := uuid.New(), uuid.New()
	fetcher := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer fetcher.Close()

	routes := new(MockSourceRouteRepository)
	handler := NewSubscriptionAggregatorHandler(service.NewIngestRSSClient(fetcher.URL, "k"), nil, routes)

	w := httptest.NewRecorder()
	handler.UnsubscribeRSS(w, routedRequest(http.MethodDelete, "/", nil, userID,
		map[string]string{"user_id": userID.String(), "feed_id": feedID.String()}))

	assert.Equal(t, http.StatusNotFound, w.Code)
	routes.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUnsubscribeRSS_MalformedFeedID(t *testing.T) {
	userID := uuid.New()
	routes := new(MockSourceRouteRepository)
	handler := NewSubscriptionAggregatorHandler(nil, nil, routes)

	w := httptest.NewRecorder()
	handler.UnsubscribeRSS(w, routedRequest(http.MethodDelete, "/", nil, userID,
		map[string]string{"user_id": userID.String(), "feed_id": "nope"}))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	routes.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// --- PUT /subscriptions/{type}/{key}/list ---

// subscriptionServers fakes the Ingest RSS and Email Ingest list endpoints.
// rssFeeds / senders are the IDs the user is currently subscribed to.
func subscriptionServers(t *testing.T, rssFeeds, senders []uuid.UUID) (*service.IngestRSSClient, *service.EmailIngestClient) {
	t.Helper()
	rss := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subs := []map[string]any{}
		for _, id := range rssFeeds {
			subs = append(subs, map[string]any{"subscription_id": uuid.NewString(), "feed_id": id.String()})
		}
		api.WriteSuccess(w, http.StatusOK, map[string]any{"subscriptions": subs, "count": len(subs)}, "v1")
	}))
	email := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := []map[string]any{}
		for _, id := range senders {
			out = append(out, map[string]any{"id": id.String(), "sender_email": "a@example.com"})
		}
		api.WriteSuccess(w, http.StatusOK, out, "v1")
	}))
	t.Cleanup(rss.Close)
	t.Cleanup(email.Close)
	return service.NewIngestRSSClient(rss.URL, "k"), service.NewEmailIngestClient(email.URL, "k")
}

func TestSetSourceList(t *testing.T) {
	userID, key := uuid.New(), uuid.New()
	params := func(typ, k string) map[string]string {
		return map[string]string{"user_id": userID.String(), "type": typ, "key": k}
	}

	t.Run("updates the route for a subscribed rss feed and email sender", func(t *testing.T) {
		rssClient, emailClient := subscriptionServers(t, []uuid.UUID{key}, []uuid.UUID{key})
		for _, typ := range []string{models.SourceTypeRSS, models.SourceTypeEmail} {
			routes := new(MockSourceRouteRepository)
			routes.On("Upsert", mock.Anything, &models.SourceRoute{
				UserID: userID, SourceType: typ, SourceKey: key, List: models.ListFeed,
			}).Return(nil)
			handler := NewSubscriptionAggregatorHandler(rssClient, emailClient, routes)

			w := httptest.NewRecorder()
			handler.SetSourceList(w, routedRequest(http.MethodPut, "/", map[string]string{"list": "feed"}, userID, params(typ, key.String())))

			assert.Equal(t, http.StatusOK, w.Code, typ)
			routes.AssertExpectations(t)
		}
	})

	t.Run("404 when the user is not subscribed to the source", func(t *testing.T) {
		// Subscribed to something else only: covers PUT-after-unsubscribe and made-up keys.
		other := uuid.New()
		rssClient, emailClient := subscriptionServers(t, []uuid.UUID{other}, []uuid.UUID{other})
		for _, typ := range []string{models.SourceTypeRSS, models.SourceTypeEmail} {
			routes := new(MockSourceRouteRepository)
			handler := NewSubscriptionAggregatorHandler(rssClient, emailClient, routes)

			w := httptest.NewRecorder()
			handler.SetSourceList(w, routedRequest(http.MethodPut, "/", map[string]string{"list": "feed"}, userID, params(typ, key.String())))

			assert.Equal(t, http.StatusNotFound, w.Code, typ)
			routes.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
		}
	})

	t.Run("500 when the subscription lookup fails", func(t *testing.T) {
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer down.Close()
		routes := new(MockSourceRouteRepository)
		handler := NewSubscriptionAggregatorHandler(service.NewIngestRSSClient(down.URL, "k"), nil, routes)

		w := httptest.NewRecorder()
		handler.SetSourceList(w, routedRequest(http.MethodPut, "/", map[string]string{"list": "feed"}, userID, params("rss", key.String())))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		routes.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
	})

	rejected := []struct {
		name       string
		authUser   uuid.UUID
		typ, key   string
		body       any
		wantStatus int
	}{
		{"other user's subscriptions", uuid.New(), "rss", key.String(), map[string]string{"list": "feed"}, http.StatusForbidden},
		{"unknown type", userID, "social", key.String(), map[string]string{"list": "feed"}, http.StatusBadRequest},
		{"malformed key", userID, "rss", "not-a-uuid", map[string]string{"list": "feed"}, http.StatusBadRequest},
		{"unknown list", userID, "rss", key.String(), map[string]string{"list": "explore"}, http.StatusBadRequest},
		{"missing list", userID, "rss", key.String(), map[string]string{}, http.StatusBadRequest},
	}
	for _, tt := range rejected {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			routes := new(MockSourceRouteRepository)
			handler := NewSubscriptionAggregatorHandler(nil, nil, routes)

			w := httptest.NewRecorder()
			handler.SetSourceList(w, routedRequest(http.MethodPut, "/", tt.body, tt.authUser, params(tt.typ, tt.key)))

			assert.Equal(t, tt.wantStatus, w.Code)
			routes.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
		})
	}

	t.Run("surfaces repository failure", func(t *testing.T) {
		rssClient, _ := subscriptionServers(t, []uuid.UUID{key}, nil)
		routes := new(MockSourceRouteRepository)
		routes.On("Upsert", mock.Anything, mock.Anything).Return(errors.New("db down"))
		handler := NewSubscriptionAggregatorHandler(rssClient, nil, routes)

		w := httptest.NewRecorder()
		handler.SetSourceList(w, routedRequest(http.MethodPut, "/", map[string]string{"list": "reads"}, userID, params("rss", key.String())))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- GET /subscriptions: each source reports its list ---

func TestListAllSubscriptions_IncludesList(t *testing.T) {
	userID := uuid.New()
	routedFeed, unroutedFeed := uuid.New(), uuid.New()

	fetcher := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteSuccess(w, http.StatusOK, map[string]any{
			"subscriptions": []map[string]any{
				{"subscription_id": uuid.NewString(), "feed_id": routedFeed.String(), "feed_title": "Routed"},
				{"subscription_id": uuid.NewString(), "feed_id": unroutedFeed.String(), "feed_title": "Unrouted"},
			},
			"count": 2,
		}, "v1")
	}))
	defer fetcher.Close()

	routes := new(MockSourceRouteRepository)
	routes.On("ListByUser", mock.Anything, userID).Return([]*models.SourceRoute{
		{UserID: userID, SourceType: models.SourceTypeRSS, SourceKey: routedFeed, List: models.ListFeed},
		// An email route sharing a feed's key must not leak onto the feed.
		{UserID: userID, SourceType: models.SourceTypeEmail, SourceKey: unroutedFeed, List: models.ListFeed},
	}, nil)
	handler := NewSubscriptionAggregatorHandler(service.NewIngestRSSClient(fetcher.URL, "k"), nil, routes)

	w := httptest.NewRecorder()
	handler.ListAllSubscriptions(w, routedRequest(http.MethodGet, "/", nil, userID, map[string]string{"user_id": userID.String()}))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp struct {
		Data struct {
			Subscriptions []struct {
				Title string `json:"title"`
				List  string `json:"list"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	got := map[string]string{}
	for _, s := range resp.Data.Subscriptions {
		got[s.Title] = s.List
	}
	assert.Equal(t, map[string]string{"Routed": "feed", "Unrouted": "reads"}, got)
}

// --- ?list= filter on list / count / search ---

func TestListFilter_PassedToRepository(t *testing.T) {
	userID := uuid.New()
	feed := models.ListFeed

	t.Run("list", func(t *testing.T) {
		ucRepo := new(MockUserContentRepository)
		ucRepo.On("ListByUserWithCursor", mock.Anything, userID, (*string)(nil), (*bool)(nil), &feed, 21, (*time.Time)(nil), (*uuid.UUID)(nil)).
			Return([]*models.UserContent{}, nil)
		contentRepo := new(MockContentRepository)
		contentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID]*models.Content{}, nil)
		handler := NewUserContentHandler(ucRepo, contentRepo, nil, nil, nil, nil)

		w := httptest.NewRecorder()
		handler.ListUserContents(w, routedRequest(http.MethodGet, "/?list=feed", nil, userID, map[string]string{"user_id": userID.String()}))

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		ucRepo.AssertExpectations(t)
	})

	t.Run("count", func(t *testing.T) {
		ucRepo := new(MockUserContentRepository)
		ucRepo.On("CountByUser", mock.Anything, userID, (*string)(nil), (*bool)(nil), &feed).Return(3, nil)
		handler := NewUserContentHandler(ucRepo, new(MockContentRepository), nil, nil, nil, nil)

		w := httptest.NewRecorder()
		handler.CountUserContents(w, routedRequest(http.MethodGet, "/count?list=feed", nil, userID, map[string]string{"user_id": userID.String()}))

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		ucRepo.AssertExpectations(t)
	})

	t.Run("search", func(t *testing.T) {
		ucRepo := new(MockUserContentRepository)
		ucRepo.On("SearchWithCursor", mock.Anything, userID, "go", &feed, 21, (*time.Time)(nil), (*uuid.UUID)(nil)).
			Return([]*models.UserContent{}, nil)
		contentRepo := new(MockContentRepository)
		contentRepo.On("GetByIDs", mock.Anything, mock.Anything).Return(map[uuid.UUID]*models.Content{}, nil)
		handler := NewUserContentHandler(ucRepo, contentRepo, nil, nil, nil, nil)

		w := httptest.NewRecorder()
		handler.SearchUserContents(w, routedRequest(http.MethodGet, "/search?q=go&list=feed", nil, userID, map[string]string{"user_id": userID.String()}))

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		ucRepo.AssertExpectations(t)
	})
}

func TestListFilter_InvalidValueRejected(t *testing.T) {
	userID := uuid.New()
	ucRepo := new(MockUserContentRepository)
	handler := NewUserContentHandler(ucRepo, new(MockContentRepository), nil, nil, nil, nil)
	params := map[string]string{"user_id": userID.String()}

	for name, call := range map[string]func(http.ResponseWriter, *http.Request){
		"list":   handler.ListUserContents,
		"count":  handler.CountUserContents,
		"search": handler.SearchUserContents,
	} {
		w := httptest.NewRecorder()
		call(w, routedRequest(http.MethodGet, "/?q=go&list=explore", nil, userID, params))
		assert.Equal(t, http.StatusBadRequest, w.Code, name)
	}
	ucRepo.AssertNotCalled(t, "ListByUserWithCursor")
	ucRepo.AssertNotCalled(t, "CountByUser")
	ucRepo.AssertNotCalled(t, "SearchWithCursor")
}

// --- PATCH accepts list (Save to Reads) ---

func TestUpdateUserContent_MovesList(t *testing.T) {
	userID, contentID, ucID := uuid.New(), uuid.New(), uuid.New()
	reads := models.ListReads

	ucRepo := new(MockUserContentRepository)
	contentRepo := new(MockContentRepository)
	ucRepo.On("GetByUserAndContent", mock.Anything, userID, contentID).
		Return(&models.UserContent{ID: ucID, UserID: userID, ContentID: contentID, List: models.ListFeed}, nil)
	ucRepo.On("UpdateMetadata", mock.Anything, ucID, (*string)(nil), (*float64)(nil), (*bool)(nil), &reads).Return(nil)
	ucRepo.On("GetByID", mock.Anything, ucID).
		Return(&models.UserContent{ID: ucID, UserID: userID, ContentID: contentID, List: models.ListReads}, nil)
	contentRepo.On("GetByID", mock.Anything, contentID).Return(&models.Content{ID: contentID}, nil)
	handler := NewUserContentHandler(ucRepo, contentRepo, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	handler.UpdateUserContent(w, routedRequest(http.MethodPatch, "/", map[string]string{"list": "reads"}, userID,
		map[string]string{"user_id": userID.String(), "content_id": contentID.String()}))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	ucRepo.AssertExpectations(t)

	var resp struct {
		Data struct {
			List string `json:"list"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, models.ListReads, resp.Data.List)
}

func TestUpdateUserContent_InvalidList(t *testing.T) {
	userID, contentID := uuid.New(), uuid.New()
	ucRepo := new(MockUserContentRepository)
	handler := NewUserContentHandler(ucRepo, new(MockContentRepository), nil, nil, nil, nil)

	w := httptest.NewRecorder()
	handler.UpdateUserContent(w, routedRequest(http.MethodPatch, "/", map[string]string{"list": "explore"}, userID,
		map[string]string{"user_id": userID.String(), "content_id": contentID.String()}))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	ucRepo.AssertNotCalled(t, "UpdateMetadata")
}
