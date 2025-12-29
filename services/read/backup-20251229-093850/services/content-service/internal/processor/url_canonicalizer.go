package processor

import (
	"net/url"
	"strings"
)

// URLCanonicalizer provides utilities for normalizing URLs
type URLCanonicalizer struct {
	trackingParams []string
}

// NewURLCanonicalizer creates a new URL canonicalizer
func NewURLCanonicalizer() *URLCanonicalizer {
	return &URLCanonicalizer{
		trackingParams: []string{
			// Google Analytics
			"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
			// Facebook
			"fbclid",
			// Google Ads
			"gclid", "gclsrc",
			// Other common tracking parameters
			"ref", "mc_cid", "mc_eid",
		},
	}
}

// Canonicalize normalizes a URL according to the following rules:
// 1. Normalize scheme (lowercase, prefer https)
// 2. Normalize host (lowercase)
// 3. Remove common tracking parameters
// 4. Consistent trailing slash handling (remove trailing slash unless it's the root path)
func (c *URLCanonicalizer) Canonicalize(rawURL string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Normalize scheme to lowercase
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)

	// Normalize host to lowercase
	parsedURL.Host = strings.ToLower(parsedURL.Host)

	// Remove tracking parameters
	c.removeTrackingParams(parsedURL)

	// Normalize trailing slash
	// Remove trailing slash unless it's the root path
	if parsedURL.Path != "/" && strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	}

	// If path is empty, set it to "/"
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	return parsedURL.String(), nil
}

// removeTrackingParams removes common tracking parameters from the URL query
func (c *URLCanonicalizer) removeTrackingParams(parsedURL *url.URL) {
	query := parsedURL.Query()

	for _, param := range c.trackingParams {
		query.Del(param)
	}

	parsedURL.RawQuery = query.Encode()
}
