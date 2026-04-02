package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
				// Allow up to 10 redirects
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
// Returns an error if:
// - The URL cannot be fetched
// - The content is too large (>5MB)
// - The content cannot be processed
func (p *ContentProcessor) ProcessURL(url string) (*ProcessedContent, error) {
	// Fetch the content
	rawHTML, err := p.fetchURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Validate content size
	if len(rawHTML) > MaxContentSize {
		return nil, fmt.Errorf("content size %d bytes exceeds maximum of %d bytes", len(rawHTML), MaxContentSize)
	}

	// Process the HTML
	return p.ProcessHTML(url, rawHTML)
}

// ProcessHTML processes raw HTML content through the pipeline
func (p *ContentProcessor) ProcessHTML(url string, rawHTML string) (*ProcessedContent, error) {
	var cleanedHTML string
	var title, author, description, imageURL string

	// Try to parse with readability
	result, err := p.readability.ParseString(url, rawHTML)
	if err != nil {
		// Fallback to raw HTML if readability fails
		cleanedHTML = rawHTML
		title = ""
		author = ""
		description = ""
	} else {
		cleanedHTML = result.Content
		title = result.Title
		author = result.Author
		description = result.Excerpt
		imageURL = result.Image
	}

	// Sanitize the HTML
	sanitizedHTML := p.sanitizer.Sanitize(cleanedHTML)

	// Generate content hash
	contentHash := p.generateContentHash(sanitizedHTML)

	// Canonicalize the URL
	canonicalURL, err := p.canonicalizer.Canonicalize(url)
	if err != nil {
		// If canonicalization fails, use the original URL
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
	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set a user agent to avoid being blocked by some sites
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; CairnBot/1.0; +https://github.com/cairn-app/cairn-reader/services/read)")

	// Execute the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body with size limit
	limitedReader := io.LimitReader(resp.Body, MaxContentSize+1)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if content exceeds size limit
	if len(bodyBytes) > MaxContentSize {
		return "", fmt.Errorf("content size exceeds maximum of %d bytes", MaxContentSize)
	}

	return string(bodyBytes), nil
}

// generateContentHash generates a SHA-256 hash of the content
func (p *ContentProcessor) generateContentHash(content string) string {
	// Normalize the content before hashing (trim whitespace, lowercase)
	normalized := strings.TrimSpace(content)

	// Generate SHA-256 hash
	hasher := sha256.New()
	hasher.Write([]byte(normalized))
	hashBytes := hasher.Sum(nil)

	// Convert to hex string
	return hex.EncodeToString(hashBytes)
}

// GetCanonicalURL returns the canonical version of a URL
func (p *ContentProcessor) GetCanonicalURL(url string) (string, error) {
	return p.canonicalizer.Canonicalize(url)
}
