package processor

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/rss/hash"
)

const (
	// MaxContentSize is the maximum size of content to process (5MB)
	MaxContentSize = 5 * 1024 * 1024 // 5MB

	// FetchTimeout is the timeout for fetching URLs
	FetchTimeout = 30 * time.Second
)

// ContentProcessor orchestrates the content processing pipeline
type ContentProcessor struct {
	readability   *ReadabilityProcessor
	sanitizer     *HTMLSanitizer
	canonicalizer *URLCanonicalizer
	httpClient    *http.Client
}

// NewContentProcessor creates a new content processor with all dependencies
func NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{
		readability:   NewReadabilityProcessor(),
		sanitizer:     NewHTMLSanitizer(),
		canonicalizer: NewURLCanonicalizer(),
		httpClient: &http.Client{
			Timeout: FetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// ProcessedContent contains the result of processing a URL
type ProcessedContent struct {
	CleanedHTML  string
	ContentHash  string
	CanonicalURL string
	Title        string
	Author       string
	Description  string
	ImageURL     string
}

// ProcessURL fetches and processes content from a URL
func (p *ContentProcessor) ProcessURL(url string) (*ProcessedContent, error) {
	rawHTML, err := p.fetchURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	return p.ProcessHTML(url, rawHTML)
}

// ProcessHTML runs readability + sanitize + hash on rawHTML. The 5MB cap is
// enforced here so the Content Service is the single source of truth for the
// limit (callers send raw HTML).
func (p *ContentProcessor) ProcessHTML(url string, rawHTML string) (*ProcessedContent, error) {
	if len(rawHTML) > MaxContentSize {
		return nil, fmt.Errorf("content size %d bytes exceeds maximum of %d bytes", len(rawHTML), MaxContentSize)
	}

	var cleanedHTML string
	var title, author, description, imageURL string

	result, err := p.readability.ParseString(url, rawHTML)
	if err != nil {
		cleanedHTML = rawHTML
	} else {
		cleanedHTML = result.Content
		title = result.Title
		author = result.Author
		description = result.Excerpt
		imageURL = result.Image
	}

	sanitizedHTML := p.sanitizer.Sanitize(cleanedHTML)

	contentHash := p.generateContentHash(sanitizedHTML)

	canonicalURL, err := p.canonicalizer.Canonicalize(url)
	if err != nil {
		canonicalURL = url
	}

	return &ProcessedContent{
		CleanedHTML:  sanitizedHTML,
		ContentHash:  contentHash,
		CanonicalURL: canonicalURL,
		Title:        title,
		Author:       author,
		Description:  description,
		ImageURL:     imageURL,
	}, nil
}

// fetchURL fetches the content from a URL with timeout and error handling
func (p *ContentProcessor) fetchURL(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CairnBot/1.0; +https://github.com/cairn-app/cairn-reader/services/read)")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, MaxContentSize+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if len(bodyBytes) > MaxContentSize {
		return "", fmt.Errorf("content size exceeds maximum of %d bytes", MaxContentSize)
	}

	return string(bodyBytes), nil
}

// generateContentHash returns the SHA-256 hash of the sanitized content.
func (p *ContentProcessor) generateContentHash(content string) string {
	return hash.ContentHash([]byte(content))
}

// GetCanonicalURL returns the canonical version of a URL
func (p *ContentProcessor) GetCanonicalURL(url string) (string, error) {
	return p.canonicalizer.Canonicalize(url)
}
