package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
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

// URLDetector defines the interface for URL detection
type URLDetector interface {
	DetectURL(ctx context.Context, url string) (*URLDetectionResult, error)
}

// urlDetectorImpl implements URLDetector
type urlDetectorImpl struct {
	httpClient *http.Client
	feedParser *gofeed.Parser
}

// NewURLDetector creates a new URL detector with a 10-second timeout
func NewURLDetector() URLDetector {
	return &urlDetectorImpl{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		feedParser: gofeed.NewParser(),
	}
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
	defer resp.Body.Close()

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
		feed, err := d.feedParser.ParseString(string(body))
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
