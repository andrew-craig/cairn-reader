package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailCleaner_TrackingPixelRemoval(t *testing.T) {
	c := NewEmailCleaner()

	tests := []struct {
		name    string
		input   string
		removed string
	}{
		{
			name:    "1x1 pixel image",
			input:   `<p>Content</p><img src="https://track.example.com/pixel.gif" width="1" height="1">`,
			removed: `width="1"`,
		},
		{
			name:    "known tracker domain - substack",
			input:   `<p>Content</p><img src="https://open.substack.com/track/abc123" width="10" height="10">`,
			removed: "open.substack.com",
		},
		{
			name:    "known tracker domain - mailchimp",
			input:   `<p>Content</p><img src="https://list-manage.com/track/open.php?u=abc" width="20" height="20">`,
			removed: "list-manage.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Clean(tt.input)
			require.NoError(t, err)
			assert.NotContains(t, result, tt.removed)
			assert.Contains(t, result, "Content")
		})
	}
}

func TestEmailCleaner_UnsubscribeRemoval(t *testing.T) {
	c := NewEmailCleaner()

	tests := []struct {
		name    string
		input   string
		removed string
	}{
		{
			name:    "unsubscribe link",
			input:   `<p>Newsletter body</p><a href="https://example.com/unsub">Unsubscribe</a>`,
			removed: "Unsubscribe",
		},
		{
			name:    "manage preferences link",
			input:   `<p>Content</p><a href="https://example.com/prefs">Manage preferences</a>`,
			removed: "Manage preferences",
		},
		{
			name:    "unsubscribe in paragraph",
			input:   `<p>Good content</p><p>Update your preferences or unsubscribe here.</p>`,
			removed: "unsubscribe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Clean(tt.input)
			require.NoError(t, err)
			assert.NotContains(t, result, tt.removed)
		})
	}
}

func TestEmailCleaner_ViewInBrowserRemoval(t *testing.T) {
	c := NewEmailCleaner()

	tests := []struct {
		name    string
		input   string
		removed string
	}{
		{
			name:    "view in browser paragraph",
			input:   `<p>View in browser</p><p>Newsletter content</p>`,
			removed: "View in browser",
		},
		{
			name:    "view this email in your browser",
			input:   `<div>View this email in your browser</div><p>Content</p>`,
			removed: "View this email in your browser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.Clean(tt.input)
			require.NoError(t, err)
			assert.NotContains(t, result, tt.removed)
		})
	}
}

func TestEmailCleaner_PreservesContent(t *testing.T) {
	c := NewEmailCleaner()

	input := `<h1>Article Title</h1>
<p>This is the main content with <a href="https://example.com">a real link</a>.</p>
<img src="https://example.com/photo.jpg" alt="photo" width="800" height="600">
<p>More content here.</p>`

	result, err := c.Clean(input)
	require.NoError(t, err)

	assert.Contains(t, result, "Article Title")
	assert.Contains(t, result, "main content")
	assert.Contains(t, result, "https://example.com")
	assert.Contains(t, result, "photo.jpg")
}

func TestEmailCleaner_HiddenPreheaderRemoval(t *testing.T) {
	c := NewEmailCleaner()

	input := `<span style="display:none;max-height:0;overflow:hidden;">Preview text for email client</span><p>Real content</p>`

	result, err := c.Clean(input)
	require.NoError(t, err)

	assert.NotContains(t, result, "Preview text for email client")
	assert.Contains(t, result, "Real content")
}
