// Package fetch provides the canonical HTTP client for Cairn RSS ingestion.
// Every outbound HTTP call originating from feed polling or full-page content
// fetches must go through Fetch so the User-Agent is consistent and connection
// pooling is shared.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// UserAgent is the canonical User-Agent for all Cairn RSS-ingestion HTTP
// traffic. It identifies the bot and provides a contact/info URL.
// Do not override this per-call; extend Fetch's options if needed.
const UserAgent = "CairnBot/1.0 (+https://github.com/andrew-craig/cairn-reader)"

const (
	defaultTimeout  = 30 * time.Second
	maxRedirects    = 10
	maxIdlePerHost  = 10
	idleConnTimeout = 90 * time.Second
	// DefaultMaxBodySize is the body-read limit applied when FetchOpts.MaxBodySize is zero.
	DefaultMaxBodySize int64 = 5 * 1024 * 1024 // 5 MB
)

// NewTransport returns an *http.Transport wired to the SSRF-safe guarded
// dialer (DialContext), with the same connection-pooling defaults Fetch
// itself uses. Callers that need an http.Client Fetch doesn't provide for
// (different redirect handling, no conditional-GET headers, a different
// timeout) must build that client's Transport from this function instead of
// constructing their own &http.Transport{} — a hand-rolled transport has no
// DialContext, so it silently loses the guard.
func NewTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConnsPerHost: maxIdlePerHost,
		IdleConnTimeout:     idleConnTimeout,
		DialContext:         DialContext,
	}
}

// sharedClient is the single http.Client used by all Fetch calls. The
// transport is tuned for concurrent multi-feed use.
var sharedClient = &http.Client{
	Timeout: defaultTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("fetch: stopped after %d redirects", maxRedirects)
		}
		return nil
	},
	Transport: NewTransport(),
}

// FetchOpts configures conditional-GET headers and resource limits for a Fetch call.
// Zero value performs an unconditional GET with the default 5 MB body limit.
type FetchOpts struct {
	// ETag is sent as If-None-Match when non-empty.
	ETag string
	// LastModified is sent as If-Modified-Since when non-empty.
	LastModified string
	// MaxBodySize caps how many bytes are read from the response body.
	// If zero, DefaultMaxBodySize (5 MB) is applied.
	MaxBodySize int64
}

// Response carries the result of a successful Fetch call.
type Response struct {
	// Body holds the response body. Empty when NotModified is true.
	Body []byte
	// ETag is the ETag header from the response, if any.
	ETag string
	// LastModified is the Last-Modified header from the response, if any.
	LastModified string
	// NotModified is true when the server returned 304 Not Modified.
	NotModified bool
	// StatusCode is the HTTP status code.
	StatusCode int
}

// Fetch performs an HTTP GET for url with the canonical User-Agent. When opts
// carries ETag/LastModified values the matching conditional headers are sent.
// Non-2xx responses other than 304 are returned as errors.
func Fetch(ctx context.Context, url string, opts FetchOpts) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch: build request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	if opts.ETag != "" {
		req.Header.Set("If-None-Match", opts.ETag)
	}
	if opts.LastModified != "" {
		req.Header.Set("If-Modified-Since", opts.LastModified)
	}

	resp, err := sharedClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	result := &Response{
		StatusCode:   resp.StatusCode,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}

	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		return result, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %q: unexpected status %d", url, resp.StatusCode)
	}

	limit := opts.MaxBodySize
	if limit <= 0 {
		limit = DefaultMaxBodySize
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("fetch %q: read body: %w", url, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("fetch %q: body exceeds limit of %d bytes", url, limit)
	}

	result.Body = body
	return result, nil
}
