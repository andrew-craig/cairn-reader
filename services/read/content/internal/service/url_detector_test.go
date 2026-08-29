package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch"
	"github.com/andrew-craig/cairn-reader/pkg/rss/fetch/fetchtest"
)

// TestNewURLDetector_UsesGuardedDialer proves the detector's HTTP client is
// wired to the shared SSRF guard (pkg/rss/fetch.DialContext) rather than a
// plain dialer that would follow caller-supplied URLs into loopback/RFC1918/
// link-local/metadata addresses.
func TestNewURLDetector_UsesGuardedDialer(t *testing.T) {
	detector, ok := NewURLDetector().(*urlDetectorImpl)
	if !ok {
		t.Fatal("expected NewURLDetector() to return *urlDetectorImpl")
	}

	transport, ok := detector.httpClient.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatal("expected httpClient.Transport to be a configured *http.Transport")
	}

	got := reflect.ValueOf(transport.DialContext).Pointer()
	want := reflect.ValueOf(fetch.DialContext).Pointer()
	if got != want {
		t.Error("expected httpClient.Transport.DialContext to be fetch.DialContext")
	}
}

func TestDetectURL_RSSFeed(t *testing.T) {
	// Create test server that returns RSS feed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Blog</title>
		<link>https://example.com</link>
		<description>A test blog</description>
		<item>
			<title>Test Article</title>
			<link>https://example.com/article</link>
		</item>
	</channel>
</rss>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	result, err := detector.DetectURL(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypeFeed {
		t.Errorf("expected type %s, got %s", URLTypeFeed, result.Type)
	}

	if result.Title == nil {
		t.Error("expected non-nil title for feed")
	} else if *result.Title != "Test Blog" {
		t.Errorf("expected title 'Test Blog', got '%s'", *result.Title)
	}
}

func TestDetectURL_AtomFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
	<title>Atom Feed Example</title>
	<link href="https://example.com"/>
	<updated>2025-01-01T00:00:00Z</updated>
	<entry>
		<title>Test Entry</title>
		<link href="https://example.com/entry"/>
	</entry>
</feed>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	result, err := detector.DetectURL(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypeFeed {
		t.Errorf("expected type %s, got %s", URLTypeFeed, result.Type)
	}

	if result.Title == nil {
		t.Error("expected non-nil title for Atom feed")
	} else if *result.Title != "Atom Feed Example" {
		t.Errorf("expected title 'Atom Feed Example', got '%s'", *result.Title)
	}
}

func TestDetectURL_HTMLPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Example Article</title>
</head>
<body>
	<h1>Example Article</h1>
	<p>This is a test article.</p>
</body>
</html>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	result, err := detector.DetectURL(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypePage {
		t.Errorf("expected type %s, got %s", URLTypePage, result.Type)
	}

	if result.Title == nil {
		t.Error("expected non-nil title for HTML page")
	} else if *result.Title != "Example Article" {
		t.Errorf("expected title 'Example Article', got '%s'", *result.Title)
	}
}

func TestDetectURL_HTMLPageNoTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<body>
	<p>No title here</p>
</body>
</html>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	result, err := detector.DetectURL(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypePage {
		t.Errorf("expected type %s, got %s", URLTypePage, result.Type)
	}

	if result.Title != nil {
		t.Errorf("expected nil title, got '%s'", *result.Title)
	}
}

func TestDetectURL_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Second) // Longer than 10s timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	detector := NewURLDetector()
	ctx, cancel := context.WithTimeout(fetchtest.AllowLoopback(context.Background()), 10*time.Second)
	defer cancel()

	result, err := detector.DetectURL(ctx, server.URL)

	if err != nil {
		t.Fatalf("DetectURL() should not return error on timeout, got %v", err)
	}

	if result.Type != URLTypeUnknown {
		t.Errorf("expected type %s on timeout, got %s", URLTypeUnknown, result.Type)
	}

	if result.Title != nil {
		t.Errorf("expected nil title on timeout, got '%s'", *result.Title)
	}
}

func TestDetectURL_InvalidURL(t *testing.T) {
	detector := NewURLDetector()
	result, err := detector.DetectURL(context.Background(), "not-a-valid-url")

	if err != nil {
		t.Fatalf("DetectURL() should not return error for invalid URL, got %v", err)
	}

	if result.Type != URLTypeUnknown {
		t.Errorf("expected type %s for invalid URL, got %s", URLTypeUnknown, result.Type)
	}

	if result.Title != nil {
		t.Errorf("expected nil title for invalid URL, got '%s'", *result.Title)
	}
}

func TestDetectURL_FeedByURL(t *testing.T) {
	// Test that feed.xml URL pattern is detected even without proper content-type
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml") // Generic XML type
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Feed by URL Pattern</title>
		<link>https://example.com</link>
	</channel>
</rss>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	// Add /feed.xml to URL to trigger URL pattern detection
	result, err := detector.DetectURL(fetchtest.AllowLoopback(context.Background()), server.URL+"/feed.xml")

	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypeFeed {
		t.Errorf("expected type %s for feed.xml URL, got %s", URLTypeFeed, result.Type)
	}
}

func TestDetectURL_Redirect(t *testing.T) {
	// Create redirect server
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Final Page</title>
</head>
<body>
	<p>After redirect</p>
</body>
</html>`))
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	detector := NewURLDetector()
	result, err := detector.DetectURL(fetchtest.AllowLoopback(context.Background()), redirectServer.URL)

	if err != nil {
		t.Fatalf("DetectURL() error = %v", err)
	}

	if result.Type != URLTypePage {
		t.Errorf("expected type %s, got %s", URLTypePage, result.Type)
	}

	// Check that final URL is returned, not redirect URL
	if result.URL != finalServer.URL {
		t.Errorf("expected URL %s, got %s", finalServer.URL, result.URL)
	}

	if result.Title == nil {
		t.Error("expected non-nil title")
	} else if *result.Title != "Final Page" {
		t.Errorf("expected title 'Final Page', got '%s'", *result.Title)
	}
}

func TestIsFeedContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{"RSS", "application/rss+xml", true},
		{"Atom", "application/atom+xml", true},
		{"XML", "application/xml", true},
		{"Text XML", "text/xml", true},
		{"HTML", "text/html", false},
		{"Plain Text", "text/plain", false},
		{"JSON", "application/json", false},
		{"RSS with charset", "application/rss+xml; charset=utf-8", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFeedContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("isFeedContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestIsFeedURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"feed path", "https://example.com/feed", true},
		{"rss path", "https://example.com/rss", true},
		{"atom path", "https://example.com/atom", true},
		{"xml extension", "https://example.com/feed.xml", true},
		{"rss extension", "https://example.com/feed.rss", true},
		{"regular page", "https://example.com/article", false},
		{"html page", "https://example.com/page.html", false},
		{"feed in subdirectory", "https://example.com/blog/feed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFeedURL(tt.url)
			if got != tt.want {
				t.Errorf("isFeedURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractHTMLTitle(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    *string
		wantNil bool
	}{
		{
			name:    "valid title",
			html:    `<!DOCTYPE html><html><head><title>Test Title</title></head></html>`,
			want:    stringPtr("Test Title"),
			wantNil: false,
		},
		{
			name:    "title with whitespace",
			html:    `<!DOCTYPE html><html><head><title>  Trimmed Title  </title></head></html>`,
			want:    stringPtr("Trimmed Title"),
			wantNil: false,
		},
		{
			name:    "no title tag",
			html:    `<!DOCTYPE html><html><head></head></html>`,
			want:    nil,
			wantNil: true,
		},
		{
			name:    "empty title",
			html:    `<!DOCTYPE html><html><head><title></title></head></html>`,
			want:    nil,
			wantNil: true,
		},
		{
			name:    "invalid html",
			html:    `not valid html`,
			want:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHTMLTitle(tt.html)
			if tt.wantNil {
				if got != nil {
					t.Errorf("extractHTMLTitle() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("extractHTMLTitle() = nil, want %v", *tt.want)
				} else if *got != *tt.want {
					t.Errorf("extractHTMLTitle() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// TestExtractHTMLTitle_DeepNestingIsBounded proves the traverse walk stops
// past maxHTMLWalkDepth rather than recursing indefinitely: a <title> tag
// nested past the bound is never found. The nesting depth (300) is kept
// under golang.org/x/net/html's own hard cap of 512 open elements (Parse
// returns a clean error beyond that), so this exercises our walk's bound
// specifically, not the parser's.
func TestExtractHTMLTitle_DeepNestingIsBounded(t *testing.T) {
	const nestDepth = 300
	html := "<html><head>" +
		strings.Repeat("<div>", nestDepth) + "<title>Buried Title</title>" + strings.Repeat("</div>", nestDepth) +
		"</head></html>"

	got := extractHTMLTitle(html)
	if got != nil {
		t.Errorf("extractHTMLTitle() = %v, want nil (title past depth bound should not be found)", *got)
	}
}

func TestDiscoverFeeds_HTMLWithLinkTags(t *testing.T) {
	// Server that serves HTML with <link rel="alternate"> feed tags
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve feeds at /feed.rss and /atom.xml
		if r.URL.Path == "/feed.rss" {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>RSS Feed</title>
		<link>https://example.com</link>
	</channel>
</rss>`))
			return
		}
		if r.URL.Path == "/atom.xml" {
			w.Header().Set("Content-Type", "application/atom+xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
	<title>Atom Feed</title>
	<link href="https://example.com"/>
</feed>`))
			return
		}

		// Default: serve HTML with links to feeds
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Example</title>
	<link rel="alternate" type="application/rss+xml" href="/feed.rss" title="RSS Feed">
	<link rel="alternate" type="application/atom+xml" href="/atom.xml" title="Atom Feed">
</head>
<body>
	<h1>Example Blog</h1>
</body>
</html>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	feeds, err := detector.DiscoverFeeds(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DiscoverFeeds() error = %v", err)
	}

	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}

	// Check that both feeds are present (order may vary)
	foundRSS := false
	foundAtom := false
	for _, feed := range feeds {
		if strings.Contains(feed.URL, "/feed.rss") && feed.Title == "RSS Feed" {
			foundRSS = true
		}
		if strings.Contains(feed.URL, "/atom.xml") && feed.Title == "Atom Feed" {
			foundAtom = true
		}
	}

	if !foundRSS {
		t.Error("expected RSS feed in results")
	}
	if !foundAtom {
		t.Error("expected Atom feed in results")
	}
}

func TestDiscoverFeeds_NoFeeds(t *testing.T) {
	// Server that serves plain HTML with no feed links and no feeds at common paths
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Example</title>
</head>
<body>
	<h1>Example Page</h1>
	<p>No feeds here.</p>
</body>
</html>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	feeds, err := detector.DiscoverFeeds(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DiscoverFeeds() error = %v", err)
	}

	if len(feeds) != 0 {
		t.Errorf("expected 0 feeds, got %d", len(feeds))
	}
}

func TestDiscoverFeeds_DirectFeed(t *testing.T) {
	// Server that serves a feed directly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Direct Feed</title>
		<link>https://example.com</link>
	</channel>
</rss>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	feeds, err := detector.DiscoverFeeds(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DiscoverFeeds() error = %v", err)
	}

	if len(feeds) != 1 {
		t.Errorf("expected 1 feed, got %d", len(feeds))
	}

	if len(feeds) > 0 && feeds[0].Title != "Direct Feed" {
		t.Errorf("expected title 'Direct Feed', got '%s'", feeds[0].Title)
	}
}

func TestDiscoverFeeds_CommonPathFallback(t *testing.T) {
	// Server that has no HTML link tags but serves a feed at /feed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Feed from /feed</title>
		<link>https://example.com</link>
	</channel>
</rss>`))
			return
		}

		// Default: serve plain HTML
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Example</title>
</head>
<body>
	<h1>Example Page</h1>
</body>
</html>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	feeds, err := detector.DiscoverFeeds(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DiscoverFeeds() error = %v", err)
	}

	if len(feeds) != 1 {
		t.Errorf("expected 1 feed, got %d", len(feeds))
	}

	if len(feeds) > 0 && feeds[0].Title != "Feed from /feed" {
		t.Errorf("expected title 'Feed from /feed', got '%s'", feeds[0].Title)
	}
}

func TestDiscoverFeeds_Deduplication(t *testing.T) {
	// Server where HTML points to /feed and /feed is also a common probe path
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Dedup Feed</title>
		<link>https://example.com</link>
	</channel>
</rss>`))
			return
		}

		// Default: serve HTML with link to /feed
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Example</title>
	<link rel="alternate" type="application/rss+xml" href="/feed">
</head>
<body>
	<h1>Example Page</h1>
</body>
</html>`))
	}))
	defer server.Close()

	detector := NewURLDetector()
	feeds, err := detector.DiscoverFeeds(fetchtest.AllowLoopback(context.Background()), server.URL)

	if err != nil {
		t.Fatalf("DiscoverFeeds() error = %v", err)
	}

	if len(feeds) != 1 {
		t.Errorf("expected 1 feed (deduplicated), got %d", len(feeds))
	}

	if len(feeds) > 0 && feeds[0].Title != "Dedup Feed" {
		t.Errorf("expected title 'Dedup Feed', got '%s'", feeds[0].Title)
	}
}

// TestExtractFeedLinks_DeepNestingIsBounded proves the traverse walk in
// extractFeedLinks stops past maxHTMLWalkDepth: a <link rel="alternate">
// nested past the bound is never found. The nesting depth (300) is kept
// under golang.org/x/net/html's own hard cap of 512 open elements (Parse
// returns a clean error beyond that), so this exercises our walk's bound
// specifically, not the parser's.
func TestExtractFeedLinks_DeepNestingIsBounded(t *testing.T) {
	const nestDepth = 300
	html := "<html><head>" +
		strings.Repeat("<div>", nestDepth) +
		`<link rel="alternate" type="application/rss+xml" href="/feed">` +
		strings.Repeat("</div>", nestDepth) +
		"</head></html>"

	d := &urlDetectorImpl{}
	feeds := d.extractFeedLinks(html, "https://example.com")

	if len(feeds) != 0 {
		t.Errorf("extractFeedLinks() = %v, want no feeds (link past depth bound should not be found)", feeds)
	}
}
