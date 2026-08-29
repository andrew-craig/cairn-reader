package processor

import (
	"fmt"
	"io"

	"github.com/andrew-craig/cairn-reader/pkg/rss/readability"
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
func (r *ReadabilityProcessor) Parse(urlStr string, htmlReader io.Reader) (*ReadabilityResult, error) {
	limitedReader := io.LimitReader(htmlReader, MaxContentSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML: %w", err)
	}
	return r.ParseString(urlStr, string(data))
}

// ParseString is a convenience method that takes an HTML string instead of a reader
func (r *ReadabilityProcessor) ParseString(urlStr string, html string) (*ReadabilityResult, error) {
	result, err := readability.Extract(urlStr, html)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML with readability: %w", err)
	}

	return &ReadabilityResult{
		Title:   result.Title,
		Author:  result.Author,
		Content: result.Content,
		Excerpt: result.Excerpt,
		Image:   result.Image,
		// TextContent, SiteName, Favicon not provided by pkg/rss/readability
	}, nil
}
