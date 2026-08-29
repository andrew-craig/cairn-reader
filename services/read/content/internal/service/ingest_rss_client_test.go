package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/api"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fetcherSubscribeResponse mirrors the flat DTO the Ingest RSS service's real
// subscribe handler returns (services/read/fetcher/internal/api/dto.SubscribeFeedResponse).
// It can't be imported directly (it lives under fetcher's internal/ tree), so this
// reproduces its JSON shape to drive the seam test off the real wire format.
type fetcherSubscribeResponse struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	FeedID         uuid.UUID `json:"feed_id"`
	FeedURL        string    `json:"feed_url"`
	FeedTitle      string    `json:"feed_title"`
	IsNewFeed      bool      `json:"is_new_feed"`
	SubscribedAt   time.Time `json:"subscribed_at"`
}

// TestSubscribeUserToFeed_UnwrapsRealServerEnvelope reproduces P2-C8: the Ingest
// RSS service returns a flat subscribe DTO wrapped in the shared pkg/api.WriteSuccess
// {data,meta} envelope (exactly as the real handler does via api.WriteSuccess), but
// the client used to unmarshal into a nested {subscription{...},feed{...}} shape that
// doesn't exist on the wire, silently "succeeding" with a zero-valued struct.
func TestSubscribeUserToFeed_UnwrapsRealServerEnvelope(t *testing.T) {
	want := fetcherSubscribeResponse{
		SubscriptionID: uuid.New(),
		FeedID:         uuid.New(),
		FeedURL:        "https://example.com/feed.xml",
		FeedTitle:      "Example Feed",
		IsNewFeed:      true,
		SubscribedAt:   time.Now().UTC().Truncate(time.Second),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteSuccess(w, http.StatusCreated, want, "v1")
	}))
	defer server.Close()

	client := NewIngestRSSClient(server.URL, "test-key")

	got, err := client.SubscribeUserToFeed(context.Background(), uuid.New().String(), want.FeedURL)
	require.NoError(t, err)

	assert.Equal(t, want.SubscriptionID.String(), got.SubscriptionID)
	assert.Equal(t, want.FeedID.String(), got.FeedID)
	assert.Equal(t, want.FeedURL, got.FeedURL)
	assert.Equal(t, want.FeedTitle, got.FeedTitle)
	assert.Equal(t, want.IsNewFeed, got.IsNewFeed)
	assert.True(t, got.SubscribedAt.Equal(want.SubscribedAt), "SubscribedAt: got %v, want %v", got.SubscribedAt, want.SubscribedAt)
}

// TestSubscribeUserToFeed_AlreadySubscribedSurfacesAsSentinel reproduces the
// 409->500 mistranslation: the real Ingest RSS subscribe handler sends the
// generic error code "conflict" (api.ErrCodeConflict) on a duplicate
// subscribe, not the "already_subscribed" code the client used to look for.
// The client must recognize the real wire shape (HTTP status, not a made-up
// code) and hand callers a distinguishable sentinel error.
func TestSubscribeUserToFeed_AlreadySubscribedSurfacesAsSentinel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "user is already subscribed to this feed", nil, "v1")
	}))
	defer server.Close()

	client := NewIngestRSSClient(server.URL, "test-key")

	_, err := client.SubscribeUserToFeed(context.Background(), uuid.New().String(), "https://example.com/feed.xml")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAlreadySubscribed), "expected errors.Is(err, ErrAlreadySubscribed), got: %v", err)
}

// TestSubscribeUserToFeed_BadRequestSurfacesAsSentinel covers the other
// status-matched branch (400): the real handler sends generic codes
// ("bad_request" for the feed-limit case, "validation_error" for an invalid
// feed) - neither of which the old client's "feed_limit_reached"/"invalid_feed"
// code matching ever recognized. The client must recognize the status alone
// and preserve the server's message.
func TestSubscribeUserToFeed_BadRequestSurfacesAsSentinel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "user has reached maximum feed limit of 100", nil, "v1")
	}))
	defer server.Close()

	client := NewIngestRSSClient(server.URL, "test-key")

	_, err := client.SubscribeUserToFeed(context.Background(), uuid.New().String(), "https://example.com/feed.xml")
	require.Error(t, err)
	var invalidReq *InvalidSubscribeRequestError
	require.True(t, errors.As(err, &invalidReq), "expected errors.As(err, *InvalidSubscribeRequestError), got: %v", err)
	assert.Equal(t, "user has reached maximum feed limit of 100", invalidReq.Msg)
}
