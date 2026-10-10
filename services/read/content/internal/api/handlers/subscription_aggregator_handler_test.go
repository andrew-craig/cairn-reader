package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/api/dto"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// sourceServer fakes a source service: healthy answers with data, otherwise 503.
func sourceServer(t *testing.T, healthy bool, data any) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		api.WriteSuccess(w, http.StatusOK, data, "v1")
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func listSubscriptions(t *testing.T, rssUp, emailUp bool) dto.ListSubscriptionsResponse {
	t.Helper()
	userID := uuid.New()
	rssURL := sourceServer(t, rssUp, map[string]any{
		"subscriptions": []map[string]any{{"id": uuid.NewString(), "feed_id": uuid.NewString(), "feed_title": "Blog"}},
	})
	emailURL := sourceServer(t, emailUp, []map[string]any{{"id": uuid.NewString(), "sender_email": "news@example.com"}})

	routes := new(MockSourceRouteRepository)
	routes.On("ListByUser", mock.Anything, userID).Return(nil, nil)
	handler := NewSubscriptionAggregatorHandler(
		service.NewIngestRSSClient(rssURL, "k"), service.NewEmailIngestClient(emailURL, "k"), routes)

	w := httptest.NewRecorder()
	handler.ListAllSubscriptions(w, routedRequest(http.MethodGet, "/", nil, userID, map[string]string{"user_id": userID.String()}))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data dto.ListSubscriptionsResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp.Data
}

func TestListAllSubscriptions_AllSourcesUp(t *testing.T) {
	resp := listSubscriptions(t, true, true)
	assert.Equal(t, 2, resp.TotalCount)
	assert.NotNil(t, resp.FailedSources)
	assert.Empty(t, resp.FailedSources)
}

func TestListAllSubscriptions_EmailDown_ReportsEmailAndKeepsRSS(t *testing.T) {
	resp := listSubscriptions(t, true, false)
	assert.Equal(t, 1, resp.TotalCount)
	assert.Equal(t, dto.SubscriptionTypeRSS, resp.Subscriptions[0].Type)
	assert.Equal(t, []dto.SubscriptionType{dto.SubscriptionTypeEmail}, resp.FailedSources)
}

func TestListAllSubscriptions_RSSDown_ReportsRSSAndKeepsEmail(t *testing.T) {
	resp := listSubscriptions(t, false, true)
	assert.Equal(t, 1, resp.TotalCount)
	assert.Equal(t, dto.SubscriptionTypeEmail, resp.Subscriptions[0].Type)
	assert.Equal(t, []dto.SubscriptionType{dto.SubscriptionTypeRSS}, resp.FailedSources)
}

func TestListAllSubscriptions_BothDown_ReportsBoth(t *testing.T) {
	resp := listSubscriptions(t, false, false)
	assert.Zero(t, resp.TotalCount)
	assert.Empty(t, resp.Subscriptions)
	assert.Equal(t, []dto.SubscriptionType{dto.SubscriptionTypeRSS, dto.SubscriptionTypeEmail}, resp.FailedSources)
}
