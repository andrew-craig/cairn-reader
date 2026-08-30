// Package sanitize provides the canonical bluemonday policy for all Cairn RSS
// ingestion. A single Policy() instance is shared across the pipeline so no
// two code paths can produce divergent sanitization behaviour.
package sanitize

import (
	htmlpkg "html"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

// singleToken matches an HTML id value: one non-whitespace token, no spaces.
// HTML5 §2.7.1 defines id as a string that contains no ASCII whitespace.
var singleToken = regexp.MustCompile(`^\S+$`)

var canonicalPolicy = buildPolicy()
var strictPolicy = bluemonday.StrictPolicy()

// buildPolicy constructs the canonical Cairn sanitizer policy.
// Source: services/read/content/internal/processor/sanitizer.go with these corrections:
//  1. <source src=…> is allowed (podcast/media enclosures)
//  2. id= is restricted to a single whitespace-free token (HTML5 §2.7.1)
//  3. <hr> is allowed (section dividers, common in newsletters and articles)
//  4. AddSpaceWhenStrippingTag inserts a space where a disallowed tag is
//     removed, so adjacent text doesn't fuse (matters for plain-text/preview
//     extraction). Both from the email service's former policy.
func buildPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AddSpaceWhenStrippingTag(true)

	// Text formatting
	p.AllowElements("p", "br", "hr", "span", "div", "article", "section")
	p.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowElements("strong", "b", "em", "i", "u", "s", "del", "ins")
	p.AllowElements("blockquote", "q", "cite", "code", "pre", "kbd", "samp", "var")

	// Lists
	p.AllowElements("ul", "ol", "li", "dl", "dt", "dd")

	// Tables (including col/colgroup for column-level styling)
	p.AllowElements("table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption", "col", "colgroup")
	p.AllowAttrs("colspan", "rowspan").OnElements("th", "td")
	p.AllowAttrs("scope").OnElements("th")

	// URL schemes must be declared before any URL-bearing attribute
	p.AllowURLSchemes("http", "https", "mailto")

	// Links
	p.AllowElements("a")
	p.AllowAttrs("href", "title").OnElements("a")
	p.RequireNoReferrerOnLinks(true)

	// Images and responsive media
	p.AllowElements("img", "figure", "figcaption", "picture", "source")
	p.AllowAttrs("src", "alt", "title").OnElements("img")
	p.AllowAttrs("width", "height").Matching(bluemonday.Integer).OnElements("img")
	// source: srcset/sizes/type/media for <picture><source>, plus src for <audio/video><source>
	p.AllowAttrs("src", "srcset", "sizes", "type", "media").OnElements("source")

	// Audio / video (podcast and rich-content feeds)
	p.AllowElements("audio", "video")
	p.AllowAttrs("src", "controls").OnElements("audio", "video")

	// class on text containers (space-separated tokens are fine here)
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements(
		"p", "div", "span", "blockquote", "code", "pre",
	)
	// id on headings: single token only (HTML5 id must contain no whitespace)
	p.AllowAttrs("id").Matching(singleToken).OnElements(
		"h1", "h2", "h3", "h4", "h5", "h6",
	)

	return p
}

// Policy returns the canonical Cairn sanitizer policy. The returned pointer is
// shared; callers must not mutate it.
func Policy() *bluemonday.Policy {
	return canonicalPolicy
}

// Sanitize runs html through the canonical policy and returns the cleaned string.
func Sanitize(html string) string {
	return canonicalPolicy.Sanitize(html)
}

// StripHTML removes all HTML tags, decodes entities, and normalises whitespace,
// returning plain text. Use this for description/preview fields.
func StripHTML(input string) string {
	if input == "" {
		return ""
	}
	text := strictPolicy.Sanitize(input)
	text = htmlpkg.UnescapeString(text)
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(text, " "))
}
