// Package readability wraps go-shiori/go-readability to extract the main
// article content from raw HTML. Readability runs exactly once per article
// ingest; do not call this on already-sanitized HTML.
package readability

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-shiori/go-readability"
)

// Result holds the extracted article metadata and cleaned content HTML.
type Result struct {
	Title   string
	Author  string
	Content string // cleaned article HTML
	Excerpt string
	Image   string
	Length  int // character length of the plain-text content
}

// Extract runs go-readability on rawHTML and returns the article content.
// rawURL is used by readability for link resolution; pass the article's
// canonical URL. Returns an error only if the URL is unparseable or
// readability fails fatally — a sparse/empty result is not an error.
func Extract(rawURL, rawHTML string) (*Result, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("readability: parse url %q: %w", rawURL, err)
	}

	article, err := readability.FromReader(strings.NewReader(rawHTML), parsedURL)
	if err != nil {
		return nil, fmt.Errorf("readability: extract %q: %w", rawURL, err)
	}

	return &Result{
		Title:   strings.TrimSpace(article.Title),
		Author:  strings.TrimSpace(article.Byline),
		Content: article.Content,
		Excerpt: strings.TrimSpace(article.Excerpt),
		Image:   strings.TrimSpace(article.Image),
		Length:  article.Length,
	}, nil
}
