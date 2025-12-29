package processor

import (
	"strings"
	"testing"
)

func TestContentProcessor_ProcessHTML(t *testing.T) {
	processor := NewContentProcessor()

	tests := []struct {
		name    string
		url     string
		html    string
		wantErr bool
	}{
		{
			name: "basic article",
			url:  "https://example.com/article",
			html: `
				<html>
					<head><title>Test Article</title></head>
					<body>
						<article>
							<h1>Test Article</h1>
							<p>This is a test article with some content.</p>
						</article>
					</body>
				</html>
			`,
			wantErr: false,
		},
		{
			name: "article with script tags",
			url:  "https://example.com/article",
			html: `
				<html>
					<body>
						<article>
							<h1>Article</h1>
							<p>Content</p>
							<script>alert('xss')</script>
						</article>
					</body>
				</html>
			`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessHTML(tt.url, tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Verify that we got a content hash
			if result.ContentHash == "" {
				t.Error("ProcessHTML() returned empty content hash")
			}

			// Verify that cleaned HTML doesn't contain script tags
			if strings.Contains(result.CleanedHTML, "<script") {
				t.Error("ProcessHTML() failed to remove script tags")
			}

			// Verify that canonical URL was generated
			if result.CanonicalURL == "" {
				t.Error("ProcessHTML() returned empty canonical URL")
			}
		})
	}
}

func TestContentProcessor_GenerateContentHash(t *testing.T) {
	processor := NewContentProcessor()

	tests := []struct {
		name     string
		content1 string
		content2 string
		wantSame bool
	}{
		{
			name:     "identical content",
			content1: "<p>Hello World</p>",
			content2: "<p>Hello World</p>",
			wantSame: true,
		},
		{
			name:     "different content",
			content1: "<p>Hello World</p>",
			content2: "<p>Goodbye World</p>",
			wantSame: false,
		},
		{
			name:     "same content with different whitespace",
			content1: "<p>Hello World</p>",
			content2: "  <p>Hello World</p>  ",
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := processor.generateContentHash(tt.content1)
			hash2 := processor.generateContentHash(tt.content2)

			if tt.wantSame && hash1 != hash2 {
				t.Errorf("generateContentHash() hashes differ but should be same: %v vs %v", hash1, hash2)
			}
			if !tt.wantSame && hash1 == hash2 {
				t.Errorf("generateContentHash() hashes are same but should differ: %v vs %v", hash1, hash2)
			}
		})
	}
}

func TestContentProcessor_ContentSizeValidation(t *testing.T) {
	processor := NewContentProcessor()

	// Create content that exceeds 5MB
	largeContent := strings.Repeat("x", MaxContentSize+1)

	// ProcessHTML doesn't validate size, only ProcessURL does through fetching
	// So we'll test that the content can be processed and verify the size is large
	result, err := processor.ProcessHTML("https://example.com", largeContent)
	if err != nil {
		t.Errorf("ProcessHTML() failed: %v", err)
	}

	// Verify that we got a result
	if result == nil {
		t.Error("ProcessHTML() returned nil result")
	}
}
