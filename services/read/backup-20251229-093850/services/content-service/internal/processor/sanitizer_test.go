package processor

import (
	"strings"
	"testing"
)

func TestHTMLSanitizer_Sanitize(t *testing.T) {
	sanitizer := NewHTMLSanitizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes script tags",
			input: "<p>Hello</p><script>alert('xss')</script>",
			want:  "<p>Hello</p>",
		},
		{
			name:  "removes onclick attributes",
			input: `<a href="https://example.com" onclick="alert('xss')">Link</a>`,
			want:  `<a href="https://example.com" rel="noreferrer">Link</a>`,
		},
		{
			name:  "keeps safe HTML",
			input: "<p>This is <strong>bold</strong> and <em>italic</em> text.</p>",
			want:  "<p>This is <strong>bold</strong> and <em>italic</em> text.</p>",
		},
		{
			name:  "keeps images with safe attributes",
			input: `<img src="https://example.com/image.jpg" alt="Test Image" width="100">`,
			want:  `<img src="https://example.com/image.jpg" alt="Test Image" width="100">`,
		},
		{
			name:  "removes dangerous iframe",
			input: `<iframe src="https://evil.com"></iframe>`,
			want:  ``,
		},
		{
			name:  "keeps lists",
			input: "<ul><li>Item 1</li><li>Item 2</li></ul>",
			want:  "<ul><li>Item 1</li><li>Item 2</li></ul>",
		},
		{
			name:  "keeps tables",
			input: `<table><thead><tr><th>Header</th></tr></thead><tbody><tr><td>Data</td></tr></tbody></table>`,
			want:  `<table><thead><tr><th>Header</th></tr></thead><tbody><tr><td>Data</td></tr></tbody></table>`,
		},
		{
			name:  "keeps safe links",
			input: `<a href="https://example.com">Link</a>`,
			want:  `<a href="https://example.com" rel="noreferrer">Link</a>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizer.Sanitize(tt.input)
			// Normalize whitespace for comparison
			got = strings.TrimSpace(got)
			want := strings.TrimSpace(tt.want)
			if got != want {
				t.Errorf("Sanitize() = %v, want %v", got, want)
			}
		})
	}
}

func TestHTMLSanitizer_XSSProtection(t *testing.T) {
	sanitizer := NewHTMLSanitizer()

	xssAttempts := []string{
		`<script>alert('XSS')</script>`,
		`<img src=x onerror="alert('XSS')">`,
		`<a href="javascript:alert('XSS')">Click</a>`,
		`<div onload="alert('XSS')">Content</div>`,
		`<iframe src="javascript:alert('XSS')"></iframe>`,
		`<object data="javascript:alert('XSS')"></object>`,
		`<embed src="javascript:alert('XSS')">`,
	}

	for _, attempt := range xssAttempts {
		t.Run("XSS_protection", func(t *testing.T) {
			result := sanitizer.Sanitize(attempt)
			// Result should not contain javascript: or any script tags
			if strings.Contains(result, "javascript:") || strings.Contains(result, "<script") {
				t.Errorf("Sanitize() failed to remove XSS attempt: %v -> %v", attempt, result)
			}
		})
	}
}
