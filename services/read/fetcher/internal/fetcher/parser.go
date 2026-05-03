package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/rss/fetch"
	"github.com/cairn-app/cairn-reader/pkg/rss/parse"
)

// ParsedFeed represents a parsed RSS/Atom feed with metadata
type ParsedFeed struct {
	Title       string
	Description string
	SiteURL     string
	Items       []*ParsedItem
}

// ParsedItem represents a single item from an RSS/Atom feed
type ParsedItem struct {
	Title       string
	Author      string
	PublishedAt *time.Time
	Description string
	URL         string
	GUID        string
}

// Parser handles RSS/Atom feed parsing
type Parser struct{}

// NewParser creates a new feed parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFromURL fetches and parses a feed from a URL
func (p *Parser) ParseFromURL(ctx context.Context, feedURL string, httpClient *http.Client) (*ParsedFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", fetch.UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, fetch.DefaultMaxBodySize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read feed body: %w", err)
	}
	if int64(len(data)) > fetch.DefaultMaxBodySize {
		return nil, fmt.Errorf("feed exceeds maximum size of %d bytes", fetch.DefaultMaxBodySize)
	}

	return p.ParseFromReader(bytes.NewReader(data))
}

// ParseFromReader parses a feed from an io.Reader
func (p *Parser) ParseFromReader(reader io.Reader) (*ParsedFeed, error) {
	feed, err := parse.ParseReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	items := make([]*ParsedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		parsed := convertParsedItem(item)
		if parsed != nil {
			items = append(items, parsed)
		}
	}

	return &ParsedFeed{
		Title:       feed.Title,
		Description: feed.Description,
		SiteURL:     feed.SiteURL,
		Items:       items,
	}, nil
}

// convertParsedItem maps a pkg/rss/parse.Item to our ParsedItem, skipping
// entries that have no usable URL or identifier.
func convertParsedItem(item parse.Item) *ParsedItem {
	// pkg/rss/parse already sets GUID = Link when the feed omits GUID.
	// If both are empty, the item has no usable identifier.
	if item.GUID == "" {
		return nil
	}

	// Prefer Link as the canonical article URL; fall back to GUID.
	url := item.Link
	if url == "" {
		url = item.GUID
	}

	return &ParsedItem{
		GUID:        item.GUID,
		Title:       item.Title,
		Author:      item.Author,
		PublishedAt: item.PublishedAt,
		Description: item.Description,
		URL:         url,
	}
}
