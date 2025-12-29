package processor

import (
	"github.com/microcosm-cc/bluemonday"
)

// HTMLSanitizer provides HTML sanitization functionality
type HTMLSanitizer struct {
	policy *bluemonday.Policy
}

// NewHTMLSanitizer creates a new HTML sanitizer with a strict policy for reading content
func NewHTMLSanitizer() *HTMLSanitizer {
	// Create a strict policy that allows common safe HTML elements
	policy := bluemonday.NewPolicy()

	// Allow common text formatting tags
	policy.AllowElements("p", "br", "span", "div", "article", "section")
	policy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowElements("strong", "b", "em", "i", "u", "s", "del", "ins")
	policy.AllowElements("blockquote", "q", "cite", "code", "pre", "kbd", "samp", "var")

	// Allow lists
	policy.AllowElements("ul", "ol", "li", "dl", "dt", "dd")

	// Allow tables
	policy.AllowElements("table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption", "col", "colgroup")
	policy.AllowAttrs("colspan", "rowspan").OnElements("th", "td")
	policy.AllowAttrs("scope").OnElements("th")

	// Allow URL schemes first (must be done before allowing URL attributes)
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
	policy.AllowAttrs("srcset", "sizes", "type", "media").OnElements("source")

	// Allow audio/video elements (for rich content)
	policy.AllowElements("audio", "video")
	policy.AllowAttrs("src").OnElements("audio", "video")
	policy.AllowAttrs("controls").OnElements("audio", "video")

	// Allow common attributes on many elements
	policy.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements(
		"p", "div", "span", "blockquote", "code", "pre",
	)
	policy.AllowAttrs("id").Matching(bluemonday.SpaceSeparatedTokens).OnElements(
		"h1", "h2", "h3", "h4", "h5", "h6",
	)

	return &HTMLSanitizer{
		policy: policy,
	}
}

// Sanitize cleans HTML content, removing potentially dangerous elements and attributes
func (s *HTMLSanitizer) Sanitize(html string) string {
	return s.policy.Sanitize(html)
}
