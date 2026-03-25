package processor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentExtractor_Extract(t *testing.T) {
	e := NewContentExtractor()

	tests := []struct {
		name            string
		input           string
		wantContains    []string
		wantNotContains []string
		wantErr         bool
	}{
		{
			name:            "script tags removed",
			input:           `<p>Hello</p><script>alert('xss')</script>`,
			wantContains:    []string{"Hello"},
			wantNotContains: []string{"<script>", "alert"},
		},
		{
			name:            "onclick attribute removed",
			input:           `<p onclick="evil()">Click me</p>`,
			wantContains:    []string{"Click me"},
			wantNotContains: []string{"onclick"},
		},
		{
			name:            "safe elements preserved",
			input:           `<h1>Title</h1><p>Body <strong>text</strong></p><ul><li>Item</li></ul>`,
			wantContains:    []string{"<h1>", "<p>", "<strong>", "<ul>", "<li>"},
			wantNotContains: []string{},
		},
		{
			name:            "links preserved",
			input:           `<a href="https://example.com" title="link">Read more</a>`,
			wantContains:    []string{`href="https://example.com"`, "Read more"},
			wantNotContains: []string{},
		},
		{
			name:            "images preserved",
			input:           `<img src="https://example.com/img.png" alt="photo" width="800" height="600">`,
			wantContains:    []string{`src="https://example.com/img.png"`, `alt="photo"`},
			wantNotContains: []string{},
		},
		{
			name:            "iframe removed",
			input:           `<p>Content</p><iframe src="evil.html"></iframe>`,
			wantContains:    []string{"Content"},
			wantNotContains: []string{"<iframe>", "evil.html"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Extract(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			for _, want := range tt.wantContains {
				assert.Contains(t, result.SanitizedHTML, want)
			}
			for _, notWant := range tt.wantNotContains {
				assert.NotContains(t, result.SanitizedHTML, notWant)
			}
		})
	}
}

func TestContentExtractor_ContentHash(t *testing.T) {
	e := NewContentExtractor()

	r1, err := e.Extract("<p>Hello world</p>")
	require.NoError(t, err)

	r2, err := e.Extract("<p>Hello world</p>")
	require.NoError(t, err)

	r3, err := e.Extract("<p>Different content</p>")
	require.NoError(t, err)

	// Same input → same hash
	assert.Equal(t, r1.ContentHash, r2.ContentHash)
	// Different input → different hash
	assert.NotEqual(t, r1.ContentHash, r3.ContentHash)
	// Hash is 64 hex chars (SHA-256)
	assert.Len(t, r1.ContentHash, 64)
}

func TestContentExtractor_PlainText(t *testing.T) {
	e := NewContentExtractor()

	result, err := e.Extract("<h1>Title</h1><p>Body text here.</p>")
	require.NoError(t, err)

	assert.NotEmpty(t, result.PlainText)
	assert.True(t, strings.Contains(result.PlainText, "Title"))
	assert.True(t, strings.Contains(result.PlainText, "Body text here"))
	assert.NotContains(t, result.PlainText, "<h1>")
	assert.NotContains(t, result.PlainText, "<p>")
}
