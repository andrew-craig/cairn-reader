package sanitizer

import (
	"testing"
)

func TestStripHTML(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "simple HTML tags",
			input:    "<p>Hello, <strong>World</strong>!</p>",
			expected: "Hello, World!",
		},
		{
			name:     "line breaks stripped",
			input:    "Hello<br />World<br/>Again",
			expected: "HelloWorldAgain",
		},
		{
			name:     "links are stripped",
			input:    `Click <a href="https://example.com">here</a> for more`,
			expected: "Click here for more",
		},
		{
			name:     "HTML entities are decoded",
			input:    "Tom &amp; Jerry &lt;3 &gt; cheese",
			expected: "Tom & Jerry <3 > cheese",
		},
		{
			name:     "multiple whitespace normalized",
			input:    "Hello    World\n\nAgain",
			expected: "Hello World Again",
		},
		{
			name:     "nested tags",
			input:    "<div><p><em>Nested</em> <strong>content</strong></p></div>",
			expected: "Nested content",
		},
		{
			name:     "image tags stripped",
			input:    `Check this <img src="photo.jpg" alt="photo"> out`,
			expected: "Check this out",
		},
		{
			name:     "script tags removed",
			input:    `Safe text<script>alert('xss')</script> more text`,
			expected: "Safe text more text",
		},
		{
			name:     "complex RSS description",
			input:    `<p>This is an article about <a href="https://example.com">technology</a>.</p><br /><p>Read more...</p>`,
			expected: "This is an article about technology.Read more...",
		},
		{
			name:     "unicode entities",
			input:    "Copyright &#169; 2024 &#8212; All rights reserved",
			expected: "Copyright © 2024 — All rights reserved",
		},
		{
			name:     "mixed content with entities",
			input:    `<p>&ldquo;Hello,&rdquo; she said. &mdash; <em>The Author</em></p>`,
			expected: "\u201cHello,\u201d she said. — The Author",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.StripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeContent(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text preserved",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "safe tags preserved",
			input:    "<p>Hello, <strong>World</strong>!</p>",
			expected: "<p>Hello, <strong>World</strong>!</p>",
		},
		{
			name:     "links preserved with noreferrer",
			input:    `<a href="https://example.com">Click here</a>`,
			expected: `<a href="https://example.com" rel="noreferrer">Click here</a>`,
		},
		{
			name:     "images preserved",
			input:    `<img src="https://example.com/photo.jpg" alt="Photo">`,
			expected: `<img src="https://example.com/photo.jpg" alt="Photo">`,
		},
		{
			name:     "script tags removed",
			input:    `<p>Safe</p><script>alert('xss')</script><p>text</p>`,
			expected: "<p>Safe</p><p>text</p>",
		},
		{
			name:     "onclick removed",
			input:    `<p onclick="alert('xss')">Click me</p>`,
			expected: "<p>Click me</p>",
		},
		{
			name:     "style tags removed",
			input:    `<style>body{color:red}</style><p>Content</p>`,
			expected: "<p>Content</p>",
		},
		{
			name:     "lists preserved",
			input:    "<ul><li>Item 1</li><li>Item 2</li></ul>",
			expected: "<ul><li>Item 1</li><li>Item 2</li></ul>",
		},
		{
			name:     "blockquote preserved",
			input:    "<blockquote>A wise quote</blockquote>",
			expected: "<blockquote>A wise quote</blockquote>",
		},
		{
			name:     "code blocks preserved",
			input:    "<pre><code>func main() {}</code></pre>",
			expected: "<pre><code>func main() {}</code></pre>",
		},
		{
			name:     "headings preserved",
			input:    "<h1>Title</h1><h2>Subtitle</h2>",
			expected: "<h1>Title</h1><h2>Subtitle</h2>",
		},
		{
			name:     "javascript URL removed",
			input:    `<a href="javascript:alert('xss')">Click</a>`,
			expected: `Click`,
		},
		{
			name:     "data URL removed",
			input:    `<img src="data:image/svg+xml,<svg onload=alert(1)>">`,
			expected: ``,
		},
		{
			name:     "iframe removed",
			input:    `<iframe src="https://evil.com"></iframe><p>Content</p>`,
			expected: "<p>Content</p>",
		},
		{
			name:     "table preserved",
			input:    "<table><tr><th>Header</th></tr><tr><td>Cell</td></tr></table>",
			expected: "<table><tr><th>Header</th></tr><tr><td>Cell</td></tr></table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeContent(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeContent(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single spaces unchanged",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "multiple spaces collapsed",
			input:    "hello    world",
			expected: "hello world",
		},
		{
			name:     "newlines collapsed",
			input:    "hello\n\nworld",
			expected: "hello world",
		},
		{
			name:     "tabs collapsed",
			input:    "hello\t\tworld",
			expected: "hello world",
		},
		{
			name:     "mixed whitespace collapsed",
			input:    "hello \n\t  world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeWhitespace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
