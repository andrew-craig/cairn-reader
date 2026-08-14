// Package processor provides email content processing functionality.
package processor

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// ProcessedEmailContent holds the result of content extraction.
type ProcessedEmailContent struct {
	SanitizedHTML string
	PlainText     string
	ContentHash   string // SHA-256 hex of sanitized HTML
}

// maxHTMLWalkDepth bounds recursive HTML tree walks so attacker-controlled
// nesting depth (e.g. millions of nested <div>s) can't stack-overflow the
// process. 100 is generous for real-world HTML.
const maxHTMLWalkDepth = 100

// ContentExtractor sanitizes email HTML and generates a content hash.
// No readability extraction — sanitize only.
type ContentExtractor struct {
	policy *bluemonday.Policy
}

// NewContentExtractor creates a ContentExtractor with a safe bluemonday policy.
func NewContentExtractor() *ContentExtractor {
	policy := bluemonday.NewPolicy()

	// Text content
	policy.AllowElements("p", "br", "hr")
	policy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowElements("strong", "em", "code", "blockquote", "pre")

	// Lists
	policy.AllowElements("ul", "ol", "li", "dl", "dt", "dd")

	// Tables
	policy.AllowElements("table", "thead", "tbody", "tr", "th", "td")
	policy.AllowAttrs("colspan", "rowspan").OnElements("th", "td")

	// Links — require nofollow + noreferrer
	policy.AllowURLSchemes("http", "https", "mailto")
	policy.AllowElements("a")
	policy.AllowAttrs("href", "title").OnElements("a")
	policy.RequireNoReferrerOnLinks(true)
	policy.AddSpaceWhenStrippingTag(true)

	// Images
	policy.AllowElements("img", "figure", "picture", "source")
	policy.AllowAttrs("src", "alt").OnElements("img")
	policy.AllowAttrs("width", "height").Matching(bluemonday.Integer).OnElements("img")

	return &ContentExtractor{policy: policy}
}

// Extract sanitizes cleanedHTML, generates a content hash, and extracts plain text.
func (e *ContentExtractor) Extract(cleanedHTML string) (ProcessedEmailContent, error) {
	sanitized := e.policy.Sanitize(cleanedHTML)

	hash := sha256.Sum256([]byte(sanitized))
	contentHash := fmt.Sprintf("%x", hash)

	plainText, err := htmlToPlainText(sanitized)
	if err != nil {
		return ProcessedEmailContent{}, fmt.Errorf("failed to extract plain text: %w", err)
	}

	return ProcessedEmailContent{
		SanitizedHTML: sanitized,
		PlainText:     plainText,
		ContentHash:   contentHash,
	}, nil
}

// htmlToPlainText walks an HTML tree and collects text nodes.
func htmlToPlainText(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	var walk func(*html.Node, int)
	walk = func(n *html.Node, depth int) {
		if depth > maxHTMLWalkDepth {
			return
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if buf.Len() > 0 {
					buf.WriteString(" ")
				}
				buf.WriteString(text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, depth+1)
		}
	}
	walk(doc, 0)

	return buf.String(), nil
}
