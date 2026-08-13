package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/rss/fetch"
	"github.com/cairn-app/cairn-reader/pkg/rss/parse"
	"golang.org/x/net/html"
)

// URLType represents the type of URL detected
type URLType string

const (
	URLTypeFeed    URLType = "feed"
	URLTypePage    URLType = "page"
	URLTypeUnknown URLType = "unknown"
)

// URLDetectionResult contains the result of URL detection
type URLDetectionResult struct {
	URL   string  `json:"url"`
	Type  URLType `json:"type"`
	Title *string `json:"title"`
}

// DiscoveredFeed represents a discovered feed from feed discovery
type DiscoveredFeed struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// URLDetector defines the interface for URL detection
type URLDetector interface {
	DetectURL(ctx context.Context, url string) (*URLDetectionResult, error)
	DiscoverFeeds(ctx context.Context, pageURL string) ([]DiscoveredFeed, error)
}

// urlDetectorImpl implements URLDetector
type urlDetectorImpl struct {
	httpClient *http.Client
}

// NewURLDetector creates a new URL detector with a 10-second timeout
func NewURLDetector() URLDetector {
	return &urlDetectorImpl{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: fetch.DialContext,
			},
		},
	}
}

// DiscoverFeeds discovers feeds from a page URL using multiple strategies
func (d *urlDetectorImpl) DiscoverFeeds(ctx context.Context, pageURL string) ([]DiscoveredFeed, error) {
	// Use 15s timeout for total discovery time
	discoveryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Step 1: Try parsing the URL directly as a feed
	if feed := d.tryParseFeed(discoveryCtx, pageURL); feed != nil {
		return []DiscoveredFeed{*feed}, nil
	}

	// Step 2: Fetch the page
	req, err := http.NewRequestWithContext(discoveryCtx, http.MethodGet, pageURL, nil)
	if err != nil {
		return []DiscoveredFeed{}, nil
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return []DiscoveredFeed{}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	// Read response body (limit to 10MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return []DiscoveredFeed{}, nil
	}

	// Step 3: Extract feed links from HTML <link rel="alternate"> tags
	feeds := d.extractFeedLinks(string(body), finalURL)

	// Step 4: Probe common paths concurrently
	baseURL, err := url.Parse(finalURL)
	if err != nil {
		// Return what we found from HTML
		return feeds, nil
	}

	commonPaths := []string{
		"/feed",
		"/rss",
		"/rss.xml",
		"/atom.xml",
		"/feed.xml",
		"/index.xml",
		"/feed/rss",
		"/feed/atom",
	}

	// Also try appending .rss to the path
	if baseURL.Path != "" {
		commonPaths = append(commonPaths, baseURL.Path+".rss")
	}

	// Worker pool for concurrent probing
	const maxWorkers = 8
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	feedsChan := make(chan *DiscoveredFeed, len(commonPaths))

	for _, path := range commonPaths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release

			// Resolve path against base URL
			var probeURL string
			if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
				probeURL = p
			} else if strings.HasPrefix(p, "/") {
				// Absolute path
				u := *baseURL
				u.Path = p
				probeURL = u.String()
			} else {
				// Relative path
				u := *baseURL
				u.Path = p
				probeURL = u.String()
			}

			if feed := d.tryParseFeed(discoveryCtx, probeURL); feed != nil {
				feedsChan <- feed
			}
		}(path)
	}

	// Close channel when all goroutines finish
	go func() {
		wg.Wait()
		close(feedsChan)
	}()

	// Collect and deduplicate results, preferring feeds with actual titles
	feedsByURL := make(map[string]*DiscoveredFeed)

	// Add feeds from HTML first
	for _, feed := range feeds {
		feedsByURL[feed.URL] = &feed
	}

	// Add/update probed feeds, preferring those with actual titles
	for feed := range feedsChan {
		existing, found := feedsByURL[feed.URL]
		if !found || existing.Title == "" {
			feedsByURL[feed.URL] = feed
		}
	}

	// Convert map to slice
	results := make([]DiscoveredFeed, 0, len(feedsByURL))
	for _, feed := range feedsByURL {
		results = append(results, *feed)
	}

	return results, nil
}

// tryParseFeed attempts to fetch and parse a URL as a feed
func (d *urlDetectorImpl) tryParseFeed(ctx context.Context, feedURL string) *DiscoveredFeed {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	// Read response body (limit to 10MB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil
	}

	feed, err := parse.ParseBytes(body)
	if err != nil || feed == nil {
		return nil
	}

	return &DiscoveredFeed{
		URL:   finalURL,
		Title: feed.Title,
	}
}

// extractFeedLinks extracts feed links from HTML <link rel="alternate"> tags
func (d *urlDetectorImpl) extractFeedLinks(htmlContent, baseURL string) []DiscoveredFeed {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return []DiscoveredFeed{}
	}

	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return []DiscoveredFeed{}
	}

	var feeds []DiscoveredFeed
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, linkType, href, title string

			// Extract attributes
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "type":
					linkType = attr.Val
				case "href":
					href = attr.Val
				case "title":
					title = attr.Val
				}
			}

			// Check if this is an alternate feed link
			if strings.Contains(rel, "alternate") &&
				(linkType == "application/rss+xml" || linkType == "application/atom+xml") &&
				href != "" {

				// Resolve relative href against base URL
				linkURL, err := url.Parse(href)
				if err != nil {
					return
				}
				resolvedURL := baseURLParsed.ResolveReference(linkURL).String()

				feeds = append(feeds, DiscoveredFeed{
					URL:   resolvedURL,
					Title: title,
				})
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return feeds
}

// DetectURL attempts to determine if a URL is a feed or a page
func (d *urlDetectorImpl) DetectURL(ctx context.Context, url string) (*URLDetectionResult, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &URLDetectionResult{
			URL:   url,
			Type:  URLTypeUnknown,
			Title: nil,
		}, nil
	}

	// Fetch URL
	resp, err := d.httpClient.Do(req)
	if err != nil {
		// On error or timeout, return unknown
		return &URLDetectionResult{
			URL:   url,
			Type:  URLTypeUnknown,
			Title: nil,
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	// Read response body (limit to 10MB to prevent memory issues)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return &URLDetectionResult{
			URL:   finalURL,
			Type:  URLTypeUnknown,
			Title: nil,
		}, nil
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")

	// Try to detect as feed first
	if isFeedContentType(contentType) || isFeedURL(finalURL) {
		feed, err := parse.ParseBytes(body)
		if err == nil && feed != nil {
			title := feed.Title
			return &URLDetectionResult{
				URL:   finalURL,
				Type:  URLTypeFeed,
				Title: &title,
			}, nil
		}
	}

	// If not a feed, try to extract page title
	if strings.Contains(contentType, "text/html") {
		title := extractHTMLTitle(string(body))
		return &URLDetectionResult{
			URL:   finalURL,
			Type:  URLTypePage,
			Title: title,
		}, nil
	}

	// Default to page with no title
	return &URLDetectionResult{
		URL:   finalURL,
		Type:  URLTypePage,
		Title: nil,
	}, nil
}

// isFeedContentType checks if the content type indicates a feed
func isFeedContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	feedTypes := []string{
		"application/rss+xml",
		"application/atom+xml",
		"application/xml",
		"text/xml",
	}

	for _, feedType := range feedTypes {
		if strings.Contains(contentType, feedType) {
			return true
		}
	}
	return false
}

// isFeedURL checks if the URL path suggests it's a feed
func isFeedURL(url string) bool {
	url = strings.ToLower(url)
	feedPatterns := []string{
		"/feed",
		"/rss",
		"/atom",
		".xml",
		".rss",
	}

	for _, pattern := range feedPatterns {
		if strings.Contains(url, pattern) {
			return true
		}
	}
	return false
}

// extractHTMLTitle extracts the title from HTML content
func extractHTMLTitle(htmlContent string) *string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	var title *string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				titleText := strings.TrimSpace(n.FirstChild.Data)
				if titleText != "" {
					title = &titleText
				}
				return
			}
			// Empty title tag - found title element but no content
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if title != nil {
				return
			}
			traverse(c)
		}
	}
	traverse(doc)

	return title
}
