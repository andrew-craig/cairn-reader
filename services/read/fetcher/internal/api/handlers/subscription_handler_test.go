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

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFeedService is a mock implementation of service.FeedService
type MockFeedService struct {
	mock.Mock
}

func (m *MockFeedService) Subscribe(ctx context.Context, userID uuid.UUID, feedURL string) (*service.SubscriptionResult, error) {
	args := m.Called(ctx, userID, feedURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SubscriptionResult), args.Error(1)
}

func (m *MockFeedService) Unsubscribe(ctx context.Context, userID, feedID uuid.UUID) error {
	args := m.Called(ctx, userID, feedID)
	return args.Error(0)
}

func (m *MockFeedService) ListUserSubscriptions(ctx context.Context, userID uuid.UUID) ([]*service.UserFeedSubscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.UserFeedSubscription), args.Error(1)
}

func (m *MockFeedService) EnableFeed(ctx context.Context, feedID uuid.UUID) error {
	args := m.Called(ctx, feedID)
	return args.Error(0)
}

func (m *MockFeedService) ValidateAndExtractFeedMetadata(ctx context.Context, feedURL string) (*service.FeedMetadata, error) {
	args := m.Called(ctx, feedURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.FeedMetadata), args.Error(1)
}

// TestSubscribe_NewFeed tests subscribing to a new feed
func TestSubscribe_NewFeed(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedID := uuid.New()
	subscriptionID := uuid.New()
	feedURL := "https://example.com/feed.xml"
	feedTitle := "Example Feed"

	result := &service.SubscriptionResult{
		Subscription: &models.FeedSubscription{
			ID:           subscriptionID,
			UserID:       userID,
			FeedID:       feedID,
			SubscribedAt: time.Now(),
		},
		Feed: &models.Feed{
			ID:      feedID,
			FeedURL: feedURL,
			Title:   &feedTitle,
			Status:  models.FeedStatusActive,
		},
		IsNewFeed: true,
	}

	mockService.On("Subscribe", mock.Anything, userID, feedURL).Return(result, nil)

	reqBody := map[string]interface{}{
		"feed_url": feedURL,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, subscriptionID.String(), response["subscription_id"])
	assert.Equal(t, feedID.String(), response["feed_id"])
	assert.Equal(t, feedURL, response["feed_url"])
	assert.Equal(t, feedTitle, response["feed_title"])
	assert.Equal(t, true, response["is_new_feed"])
	mockService.AssertExpectations(t)
}

// TestSubscribe_ExistingFeed tests subscribing to an existing feed
func TestSubscribe_ExistingFeed(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedID := uuid.New()
	subscriptionID := uuid.New()
	feedURL := "https://example.com/feed.xml"
	feedTitle := "Example Feed"

	result := &service.SubscriptionResult{
		Subscription: &models.FeedSubscription{
			ID:           subscriptionID,
			UserID:       userID,
			FeedID:       feedID,
			SubscribedAt: time.Now(),
		},
		Feed: &models.Feed{
			ID:      feedID,
			FeedURL: feedURL,
			Title:   &feedTitle,
			Status:  models.FeedStatusActive,
		},
		IsNewFeed: false, // Existing feed
	}

	mockService.On("Subscribe", mock.Anything, userID, feedURL).Return(result, nil)

	reqBody := map[string]interface{}{
		"feed_url": feedURL,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, false, response["is_new_feed"])
	mockService.AssertExpectations(t)
}

// TestSubscribe_InvalidUserID tests handling of invalid user ID
func TestSubscribe_InvalidUserID(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	reqBody := map[string]interface{}{
		"feed_url": "https://example.com/feed.xml",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/invalid-uuid/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_user_id", response["error"])
}

// TestSubscribe_InvalidJSON tests handling of invalid JSON
func TestSubscribe_InvalidJSON(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_request", response["error"])
}

// TestSubscribe_EmptyFeedURL tests handling of empty feed URL
func TestSubscribe_EmptyFeedURL(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()

	reqBody := map[string]interface{}{
		"feed_url": "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_feed_url", response["error"])
}

// TestSubscribe_FeedLimitExceeded tests handling when user reaches 100 feed limit
func TestSubscribe_FeedLimitExceeded(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedURL := "https://example.com/feed.xml"

	mockService.On("Subscribe", mock.Anything, userID, feedURL).
		Return(nil, errors.New("maximum feed limit of 100 reached"))

	reqBody := map[string]interface{}{
		"feed_url": feedURL,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "feed_limit_reached", response["error"])
	mockService.AssertExpectations(t)
}

// TestSubscribe_AlreadySubscribed tests handling when user is already subscribed
func TestSubscribe_AlreadySubscribed(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedURL := "https://example.com/feed.xml"

	mockService.On("Subscribe", mock.Anything, userID, feedURL).
		Return(nil, errors.New("already subscribed to this feed"))

	reqBody := map[string]interface{}{
		"feed_url": feedURL,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "already_subscribed", response["error"])
	mockService.AssertExpectations(t)
}

// TestSubscribe_InvalidFeedURL tests handling when feed URL is not valid RSS/Atom
func TestSubscribe_InvalidFeedURL(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedURL := "https://example.com/not-a-feed"

	mockService.On("Subscribe", mock.Anything, userID, feedURL).
		Return(nil, errors.New("failed to validate feed: invalid format"))

	reqBody := map[string]interface{}{
		"feed_url": feedURL,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/feeds/subscribe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Subscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_feed", response["error"])
	mockService.AssertExpectations(t)
}

// TestUnsubscribe_Success tests successful unsubscription
func TestUnsubscribe_Success(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedID := uuid.New()

	mockService.On("Unsubscribe", mock.Anything, userID, feedID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/feeds/"+feedID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("feed_id", feedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Unsubscribe(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, true, response["success"])
	assert.Contains(t, response["message"], "Successfully unsubscribed")
	mockService.AssertExpectations(t)
}

// TestUnsubscribe_InvalidUserID tests handling of invalid user ID
func TestUnsubscribe_InvalidUserID(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	feedID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/invalid-uuid/feeds/"+feedID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", "invalid-uuid")
	rctx.URLParams.Add("feed_id", feedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Unsubscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_user_id", response["error"])
}

// TestUnsubscribe_InvalidFeedID tests handling of invalid feed ID
func TestUnsubscribe_InvalidFeedID(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/feeds/invalid-uuid", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("feed_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Unsubscribe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_feed_id", response["error"])
}

// TestUnsubscribe_NotSubscribed tests handling when subscription doesn't exist
func TestUnsubscribe_NotSubscribed(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedID := uuid.New()

	mockService.On("Unsubscribe", mock.Anything, userID, feedID).
		Return(errors.New("subscription not found"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String()+"/feeds/"+feedID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	rctx.URLParams.Add("feed_id", feedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.Unsubscribe(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "subscription_not_found", response["error"])
	mockService.AssertExpectations(t)
}

// TestListSubscriptions_Success tests successful listing of subscriptions
func TestListSubscriptions_Success(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()
	feedID1 := uuid.New()
	feedID2 := uuid.New()
	feedTitle1 := "Feed 1"
	feedTitle2 := "Feed 2"

	subscriptions := []*service.UserFeedSubscription{
		{
			Subscription: &models.FeedSubscription{
				ID:           uuid.New(),
				UserID:       userID,
				FeedID:       feedID1,
				SubscribedAt: time.Now(),
			},
			Feed: &models.Feed{
				ID:          feedID1,
				FeedURL:     "https://example.com/feed1.xml",
				Title:       &feedTitle1,
				Status:      models.FeedStatusActive,
				PollingTier: "active",
			},
		},
		{
			Subscription: &models.FeedSubscription{
				ID:           uuid.New(),
				UserID:       userID,
				FeedID:       feedID2,
				SubscribedAt: time.Now(),
			},
			Feed: &models.Feed{
				ID:          feedID2,
				FeedURL:     "https://example.com/feed2.xml",
				Title:       &feedTitle2,
				Status:      models.FeedStatusActive,
				PollingTier: "moderate",
			},
		},
	}

	mockService.On("ListUserSubscriptions", mock.Anything, userID).Return(subscriptions, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/feeds", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListSubscriptions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, float64(2), response["count"])
	subs := response["subscriptions"].([]interface{})
	assert.Len(t, subs, 2)
	mockService.AssertExpectations(t)
}

// TestListSubscriptions_Empty tests listing when user has no subscriptions
func TestListSubscriptions_Empty(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	userID := uuid.New()

	mockService.On("ListUserSubscriptions", mock.Anything, userID).Return([]*service.UserFeedSubscription{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/feeds", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListSubscriptions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, float64(0), response["count"])
	mockService.AssertExpectations(t)
}

// TestListSubscriptions_InvalidUserID tests handling of invalid user ID
func TestListSubscriptions_InvalidUserID(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid/feeds", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.ListSubscriptions(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_user_id", response["error"])
}

// TestEnableFeed_Success tests successfully re-enabling a disabled feed
func TestEnableFeed_Success(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	feedID := uuid.New()

	mockService.On("EnableFeed", mock.Anything, feedID).Return(nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/feeds/"+feedID.String()+"/enable", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feed_id", feedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.EnableFeed(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, true, response["success"])
	assert.Contains(t, response["message"], "successfully enabled")
	mockService.AssertExpectations(t)
}

// TestEnableFeed_InvalidFeedID tests handling of invalid feed ID
func TestEnableFeed_InvalidFeedID(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/feeds/invalid-uuid/enable", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feed_id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.EnableFeed(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "invalid_feed_id", response["error"])
}

// TestEnableFeed_NotFound tests handling when feed doesn't exist
func TestEnableFeed_NotFound(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	feedID := uuid.New()

	mockService.On("EnableFeed", mock.Anything, feedID).
		Return(errors.New("feed not found"))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/feeds/"+feedID.String()+"/enable", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feed_id", feedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.EnableFeed(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "feed_not_found", response["error"])
	mockService.AssertExpectations(t)
}

// TestEnableFeed_AlreadyActive tests handling when feed is already active
func TestEnableFeed_AlreadyActive(t *testing.T) {
	mockService := new(MockFeedService)
	handler := NewSubscriptionHandler(mockService)

	feedID := uuid.New()

	mockService.On("EnableFeed", mock.Anything, feedID).
		Return(errors.New("feed is already active"))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/feeds/"+feedID.String()+"/enable", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feed_id", feedID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.EnableFeed(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	assert.Equal(t, "already_active", response["error"])
	mockService.AssertExpectations(t)
}
