// Package processor provides email content processing functionality.
package processor

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/andrew-craig/cairn-reader/pkg/rss/sanitize"
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
type ContentExtractor struct{}

// NewContentExtractor creates a ContentExtractor.
func NewContentExtractor() *ContentExtractor {
	return &ContentExtractor{}
}

// Extract sanitizes cleanedHTML, generates a content hash, and extracts plain text.
// Sanitization uses the canonical pkg/rss/sanitize policy, the same policy the
// content service applies to RSS-sourced HTML — email is an untrusted inbound
// path and must not diverge from it.
func (e *ContentExtractor) Extract(cleanedHTML string) (ProcessedEmailContent, error) {
	sanitized := sanitize.Sanitize(cleanedHTML)

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
