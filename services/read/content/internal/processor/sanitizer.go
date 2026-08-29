package processor

import (
	"github.com/andrew-craig/cairn-reader/pkg/rss/sanitize"
)

// HTMLSanitizer provides HTML sanitization functionality
type HTMLSanitizer struct{}

// NewHTMLSanitizer creates a new HTML sanitizer using the canonical pkg/rss policy
func NewHTMLSanitizer() *HTMLSanitizer {
	return &HTMLSanitizer{}
}

// Sanitize cleans HTML content, removing potentially dangerous elements and attributes
func (s *HTMLSanitizer) Sanitize(html string) string {
	return sanitize.Sanitize(html)
}
