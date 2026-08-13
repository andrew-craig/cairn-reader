package fetch_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cairn-app/cairn-reader/pkg/rss/fetch"
	"github.com/cairn-app/cairn-reader/pkg/rss/fetch/fetchtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserAgentConstant(t *testing.T) {
	// The constant must be non-empty and contain the bot name.
	assert.NotEmpty(t, fetch.UserAgent)
	assert.Contains(t, fetch.UserAgent, "CairnBot")
	assert.Contains(t, fetch.UserAgent, "https://github.com/cairn-app/cairn-reader")
}

func TestFetch_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		fmt.Fprint(w, "body")
	}))
	defer srv.Close()

	resp, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{})
	require.NoError(t, err)
	assert.Equal(t, fetch.UserAgent, gotUA)
	assert.Equal(t, []byte("body"), resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFetch_ConditionalGet_ETag(t *testing.T) {
	const etag = `"abc123"`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		fmt.Fprint(w, "content")
	}))
	defer srv.Close()

	// First fetch — no ETag yet
	resp, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{})
	require.NoError(t, err)
	assert.False(t, resp.NotModified)
	assert.Equal(t, etag, resp.ETag)

	// Second fetch — sends ETag, gets 304
	resp2, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{ETag: resp.ETag})
	require.NoError(t, err)
	assert.True(t, resp2.NotModified)
	assert.Equal(t, http.StatusNotModified, resp2.StatusCode)
	assert.Empty(t, resp2.Body)
}

func TestFetch_ConditionalGet_LastModified(t *testing.T) {
	const lastMod = "Mon, 02 Jan 2006 15:04:05 GMT"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Modified-Since") == lastMod {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Last-Modified", lastMod)
		fmt.Fprint(w, "content")
	}))
	defer srv.Close()

	resp, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{})
	require.NoError(t, err)
	assert.Equal(t, lastMod, resp.LastModified)

	resp2, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{LastModified: resp.LastModified})
	require.NoError(t, err)
	assert.True(t, resp2.NotModified)
}

func TestFetch_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFetch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{})
	assert.Error(t, err)
}

func TestFetch_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond — context should cancel first
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(fetchtest.AllowLoopback(context.Background()))
	cancel() // cancel immediately

	_, err := fetch.Fetch(ctx, srv.URL, fetch.FetchOpts{})
	assert.Error(t, err)
}

func TestFetch_InvalidURL(t *testing.T) {
	_, err := fetch.Fetch(context.Background(), "://not-a-url", fetch.FetchOpts{})
	assert.Error(t, err)
}

func TestFetch_BodyExceedsLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 11 bytes; limit is set to 10
		fmt.Fprint(w, "01234567890")
	}))
	defer srv.Close()

	_, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{MaxBodySize: 10})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds limit")
}

func TestFetch_BodyAtLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "0123456789") // exactly 10 bytes
	}))
	defer srv.Close()

	resp, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{MaxBodySize: 10})
	require.NoError(t, err)
	assert.Equal(t, 10, len(resp.Body))
}

func TestFetch_DefaultLimitApplied(t *testing.T) {
	// Verify MaxBodySize=0 uses the default (does not panic, succeeds for small body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "small")
	}))
	defer srv.Close()

	resp, err := fetch.Fetch(fetchtest.AllowLoopback(context.Background()), srv.URL, fetch.FetchOpts{})
	require.NoError(t, err)
	assert.Equal(t, []byte("small"), resp.Body)
}
