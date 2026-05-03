// Package parse wraps gofeed to produce canonical Feed and Item structs for
// RSS/Atom ingestion. All author resolution and field normalization happens here
// so callers never touch gofeed types directly.
package parse

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// Feed is the canonical representation of a parsed RSS/Atom feed.
type Feed struct {
	Title       string
	SiteURL     string
	Description string
	Items       []Item
}

// Item is the canonical representation of a single feed entry.
type Item struct {
	// GUID is the feed-supplied unique identifier (falls back to Link when absent).
	GUID string
	// Title is the entry title with leading/trailing whitespace trimmed.
	Title string
	// Author is resolved in priority order: RSS/Atom <author> → dc:creator → "".
	// Email-only authors (no display name) are treated as absent.
	Author string
	// Link is the canonical URL for the article.
	Link string
	// PublishedAt is nil when the feed provides no date.
	PublishedAt *time.Time
	// Description is the raw RSS <description> / Atom <summary> field.
	Description string
	// Content is <content:encoded> when present, otherwise Description.
	Content string
}

// ParseBytes parses an RSS/Atom feed from a byte slice.
func ParseBytes(data []byte) (*Feed, error) {
	return ParseReader(bytes.NewReader(data))
}

// ParseReader parses an RSS/Atom feed from an io.Reader.
func ParseReader(r io.Reader) (*Feed, error) {
	p := gofeed.NewParser()
	gf, err := p.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}

	feed := &Feed{
		Title:       strings.TrimSpace(gf.Title),
		Description: strings.TrimSpace(gf.Description),
		Items:       make([]Item, 0, len(gf.Items)),
	}

	// Prefer the feed's declared link; fall back to FeedLink (self-link).
	if gf.Link != "" {
		feed.SiteURL = gf.Link
	} else {
		feed.SiteURL = gf.FeedLink
	}

	for _, gi := range gf.Items {
		feed.Items = append(feed.Items, convertItem(gi))
	}

	return feed, nil
}

// convertItem maps a gofeed.Item to our canonical Item type.
func convertItem(gi *gofeed.Item) Item {
	guid := gi.GUID
	if guid == "" {
		guid = gi.Link
	}

	content := gi.Content
	if content == "" {
		content = gi.Description
	}

	item := Item{
		GUID:        guid,
		Title:       strings.TrimSpace(gi.Title),
		Author:      resolveAuthor(gi),
		Link:        gi.Link,
		PublishedAt: resolvePublishedAt(gi),
		Description: gi.Description,
		Content:     content,
	}
	return item
}

// resolveAuthor extracts a display-name author using this priority chain:
//  1. gi.Author.Name (gofeed normalizes RSS <author> and Atom <author><name> here)
//  2. gi.Authors[0].Name (Atom multi-author feeds)
//  3. dc:creator via DublinCoreExt
//  4. "" (email-only authors and feeds that omit author entirely)
func resolveAuthor(gi *gofeed.Item) string {
	if gi.Author != nil && gi.Author.Name != "" {
		return strings.TrimSpace(gi.Author.Name)
	}
	if len(gi.Authors) > 0 && gi.Authors[0] != nil && gi.Authors[0].Name != "" {
		return strings.TrimSpace(gi.Authors[0].Name)
	}
	if gi.DublinCoreExt != nil && len(gi.DublinCoreExt.Creator) > 0 {
		if name := strings.TrimSpace(gi.DublinCoreExt.Creator[0]); name != "" {
			return name
		}
	}
	return ""
}

// resolvePublishedAt returns the best available publication timestamp.
// PublishedParsed is preferred; UpdatedParsed is the fallback; nil when absent.
func resolvePublishedAt(gi *gofeed.Item) *time.Time {
	if gi.PublishedParsed != nil {
		return gi.PublishedParsed
	}
	return gi.UpdatedParsed
}
