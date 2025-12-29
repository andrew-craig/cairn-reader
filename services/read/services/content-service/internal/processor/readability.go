package processor

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-shiori/go-readability"
)

// ReadabilityProcessor provides HTML content extraction and cleaning
type ReadabilityProcessor struct{}

// NewReadabilityProcessor creates a new readability processor
func NewReadabilityProcessor() *ReadabilityProcessor {
	return &ReadabilityProcessor{}
}

// ReadabilityResult contains the extracted content and metadata
type ReadabilityResult struct {
	Title       string
	Author      string
	Content     string
	TextContent string
	Excerpt     string
	SiteName    string
	Image       string
	Favicon     string
}

// Parse processes HTML content to extract the main article content
// Falls back to the raw HTML if readability parsing fails
func (r *ReadabilityProcessor) Parse(urlStr string, htmlReader io.Reader) (*ReadabilityResult, error) {
	// Parse the URL string into a *url.URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Parse the HTML using go-readability
	article, err := readability.FromReader(htmlReader, parsedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML with readability: %w", err)
	}

	// Extract the author, handling empty strings
	author := strings.TrimSpace(article.Byline)

	return &ReadabilityResult{
		Title:       strings.TrimSpace(article.Title),
		Author:      author,
		Content:     article.Content,
		TextContent: strings.TrimSpace(article.TextContent),
		Excerpt:     strings.TrimSpace(article.Excerpt),
		SiteName:    strings.TrimSpace(article.SiteName),
		Image:       strings.TrimSpace(article.Image),
		Favicon:     strings.TrimSpace(article.Favicon),
	}, nil
}

// ParseString is a convenience method that takes an HTML string instead of a reader
func (r *ReadabilityProcessor) ParseString(url string, html string) (*ReadabilityResult, error) {
	return r.Parse(url, strings.NewReader(html))
}
