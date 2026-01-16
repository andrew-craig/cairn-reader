// Package sanitizer provides HTML sanitization for RSS feed content.
// It removes or sanitizes HTML tags from article descriptions and content
// to ensure clean, safe text for display.
package sanitizer

import (
	"html"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// HTMLSanitizer provides HTML sanitization functionality for RSS content.
type HTMLSanitizer struct {
	// stripPolicy removes all HTML tags, leaving only text
	stripPolicy *bluemonday.Policy
	// contentPolicy allows safe HTML elements for rich content display
	contentPolicy *bluemonday.Policy
}

// New creates a new HTML sanitizer with policies for stripping and sanitizing HTML.
func New() *HTMLSanitizer {
	return &HTMLSanitizer{
		stripPolicy:   createStripPolicy(),
		contentPolicy: createContentPolicy(),
	}
}

// createStripPolicy creates a policy that removes all HTML tags.
// Used for description fields where we want plain text only.
func createStripPolicy() *bluemonday.Policy {
	return bluemonday.StrictPolicy()
}

// createContentPolicy creates a policy that allows safe HTML elements.
// Used for content fields where we want to preserve formatting.
func createContentPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()

	// Allow common text formatting tags
	policy.AllowElements("p", "br", "span", "div", "article", "section")
	policy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowElements("strong", "b", "em", "i", "u", "s", "del", "ins")
	policy.AllowElements("blockquote", "q", "cite", "code", "pre", "kbd", "samp", "var")

	// Allow lists
	policy.AllowElements("ul", "ol", "li", "dl", "dt", "dd")

	// Allow tables
	policy.AllowElements("table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption")
	policy.AllowAttrs("colspan", "rowspan").OnElements("th", "td")
	policy.AllowAttrs("scope").OnElements("th")

	// Allow URL schemes (must be done before allowing URL attributes)
	policy.AllowURLSchemes("http", "https", "mailto")

	// Allow links with safe attributes
	policy.AllowElements("a")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("title").OnElements("a")
	policy.RequireNoReferrerOnLinks(true)

	// Allow images with safe attributes
	policy.AllowElements("img", "figure", "figcaption", "picture", "source")
	policy.AllowAttrs("src").OnElements("img")
	policy.AllowAttrs("alt", "title").OnElements("img")
	policy.AllowAttrs("width", "height").Matching(bluemonday.Integer).OnElements("img")

	return policy
}

// StripHTML removes all HTML tags and returns plain text.
// Also decodes HTML entities and normalizes whitespace.
// Use this for description/preview text where HTML is not wanted.
func (s *HTMLSanitizer) StripHTML(input string) string {
	if input == "" {
		return ""
	}

	// First, strip all HTML tags
	text := s.stripPolicy.Sanitize(input)

	// Decode HTML entities (e.g., &amp; -> &, &lt; -> <)
	text = html.UnescapeString(text)

	// Normalize whitespace: collapse multiple spaces/newlines into single space
	text = normalizeWhitespace(text)

	return strings.TrimSpace(text)
}

// SanitizeContent sanitizes HTML content, allowing safe tags while removing dangerous ones.
// Use this for full article content where formatting should be preserved.
func (s *HTMLSanitizer) SanitizeContent(input string) string {
	if input == "" {
		return ""
	}

	return s.contentPolicy.Sanitize(input)
}

// whitespaceRegex matches one or more whitespace characters (spaces, tabs, newlines)
var whitespaceRegex = regexp.MustCompile(`\s+`)

// normalizeWhitespace collapses multiple whitespace characters into a single space.
func normalizeWhitespace(input string) string {
	return whitespaceRegex.ReplaceAllString(input, " ")
}
