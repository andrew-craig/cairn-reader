package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
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
type Parser struct {
	parser *gofeed.Parser
}

// NewParser creates a new feed parser
func NewParser() *Parser {
	return &Parser{
		parser: gofeed.NewParser(),
	}
}

// ParseFromURL fetches and parses a feed from a URL
func (p *Parser) ParseFromURL(ctx context.Context, feedURL string, httpClient *http.Client) (*ParsedFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Cairn-RSS-Fetcher/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return p.ParseFromReader(resp.Body)
}

// ParseFromReader parses a feed from an io.Reader
func (p *Parser) ParseFromReader(reader io.Reader) (*ParsedFeed, error) {
	feed, err := p.parser.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Extract feed metadata
	title := ""
	if feed.Title != "" {
		title = feed.Title
	}

	description := ""
	if feed.Description != "" {
		description = feed.Description
	}

	siteURL := ""
	if feed.Link != "" {
		siteURL = feed.Link
	}

	// Parse items
	items := make([]*ParsedItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		parsedItem := p.parseItem(item)
		if parsedItem != nil {
			items = append(items, parsedItem)
		}
	}

	return &ParsedFeed{
		Title:       title,
		Description: description,
		SiteURL:     siteURL,
		Items:       items,
	}, nil
}

// parseItem converts a gofeed.Item to our ParsedItem
func (p *Parser) parseItem(item *gofeed.Item) *ParsedItem {
	if item == nil {
		return nil
	}

	// Extract GUID (prefer GUID, fallback to Link)
	guid := ""
	if item.GUID != "" {
		guid = item.GUID
	} else if item.Link != "" {
		guid = item.Link
	} else {
		// Skip items without any identifier
		return nil
	}

	// Extract URL (prefer Link, fallback to GUID if it looks like a URL)
	url := ""
	if item.Link != "" {
		url = item.Link
	} else if item.GUID != "" {
		url = item.GUID
	}

	if url == "" {
		// Skip items without a URL
		return nil
	}

	// Extract title
	title := ""
	if item.Title != "" {
		title = item.Title
	}

	// Extract author
	author := ""
	if item.Author != nil && item.Author.Name != "" {
		author = item.Author.Name
	}

	// Extract published date
	var publishedAt *time.Time
	if item.PublishedParsed != nil {
		publishedAt = item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		publishedAt = item.UpdatedParsed
	}

	// Extract description (prefer Description, fallback to Content)
	description := ""
	if item.Description != "" {
		description = item.Description
	} else if item.Content != "" {
		description = item.Content
	}

	return &ParsedItem{
		Title:       title,
		Author:      author,
		PublishedAt: publishedAt,
		Description: description,
		URL:         url,
		GUID:        guid,
	}
}
