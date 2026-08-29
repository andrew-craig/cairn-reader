package fetch_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch"
	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch/fetchtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch_BlocksAddressRanges(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"loopback IPv4", "http://127.0.0.1:80/"},
		{"loopback IPv4 alt", "http://127.1.2.3:80/"},
		{"loopback IPv6", "http://[::1]:80/"},
		{"RFC1918 10.0.0.0/8", "http://10.1.2.3/"},
		{"RFC1918 172.16.0.0/12", "http://172.16.5.6/"},
		{"RFC1918 192.168.0.0/16", "http://192.168.1.1/"},
		{"link-local", "http://169.254.1.1/"},
		{"cloud metadata", "http://169.254.169.254/"},
		{"IPv6 link-local", "http://[fe80::1]/"},
		{"IPv6 unique-local", "http://[fc00::1]/"},
		{"unspecified IPv4", "http://0.0.0.0/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fetch.Fetch(context.Background(), tt.url, fetch.FetchOpts{})
			require.Error(t, err)
			assert.ErrorIs(t, err, fetch.ErrBlockedAddress)
		})
	}
}

func TestFetch_BlocksRedirectToInternalAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer srv.Close()

	ctx := fetchtest.AllowLoopback(context.Background())
	_, err := fetch.Fetch(ctx, srv.URL, fetch.FetchOpts{})
	require.Error(t, err)
	assert.ErrorIs(t, err, fetch.ErrBlockedAddress)
}

func TestFetch_BlocksHostnameResolvingToInternalIP(t *testing.T) {
	resolver := fetchtest.FakeResolver{
		"internal.example.test": {net.ParseIP("169.254.169.254")},
	}
	ctx := fetchtest.WithResolver(context.Background(), resolver)

	_, err := fetch.Fetch(ctx, "http://internal.example.test/", fetch.FetchOpts{})
	require.Error(t, err)
	assert.ErrorIs(t, err, fetch.ErrBlockedAddress)
}

func TestFetch_BlocksHostnameWithAnyResolvedIPInternal(t *testing.T) {
	// DNS can return multiple addresses; if any one of them is internal the
	// whole lookup must be refused rather than silently dialing whichever
	// address happens to look safe.
	resolver := fetchtest.FakeResolver{
		"mixed.example.test": {net.ParseIP("93.184.216.34"), net.ParseIP("10.0.0.1")},
	}
	ctx := fetchtest.WithResolver(context.Background(), resolver)

	_, err := fetch.Fetch(ctx, "http://mixed.example.test/", fetch.FetchOpts{})
	require.Error(t, err)
	assert.ErrorIs(t, err, fetch.ErrBlockedAddress)
}

func TestFetch_AllowsHostnameResolvingToPublicIP(t *testing.T) {
	// A hostname resolving only to public addresses (here, the real address
	// of a local httptest server, reached via loopback-allowance) must still
	// succeed — the guard should not over-block.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)

	resolver := fetchtest.FakeResolver{"public.example.test": {net.ParseIP(host)}}
	ctx := fetchtest.WithResolver(fetchtest.AllowLoopback(context.Background()), resolver)

	resp, err := fetch.Fetch(ctx, "http://public.example.test:"+port+"/", fetch.FetchOpts{})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
