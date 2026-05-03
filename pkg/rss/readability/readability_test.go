package readability_test

import (
	"strings"
	"testing"

	"github.com/cairn-app/cairn-reader/pkg/rss/readability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalArticleHTML is a small but realistic article page.
const minimalArticleHTML = `<!DOCTYPE html>
<html>
<head><title>Go Modules Explained</title></head>
<body>
  <header><nav>Site Nav</nav></header>
  <main>
    <article>
      <h1>Go Modules Explained</h1>
      <p class="byline">By Jane Doe</p>
      <p>Go modules are the standard dependency management system for Go programs.
      They were introduced in Go 1.11 and became the default in Go 1.16.</p>
      <p>A module is a collection of related packages versioned together.
      The module path, declared in go.mod, identifies the module uniquely.</p>
      <p>Dependencies are recorded in go.mod and verified by checksums in go.sum.
      This makes builds reproducible and secure across different environments.</p>
    </article>
  </main>
  <footer>Footer content</footer>
</body>
</html>`

func TestExtract_HappyPath(t *testing.T) {
	result, err := readability.Extract("https://example.com/go-modules", minimalArticleHTML)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Title should be extracted
	assert.NotEmpty(t, result.Title)
	// Content should contain the article body
	assert.Contains(t, result.Content, "Go modules")
	// Length should be positive
	assert.Greater(t, result.Length, 0)
}

func TestExtract_InvalidURL(t *testing.T) {
	_, err := readability.Extract("://not-a-url", "<html><body>text</body></html>")
	assert.Error(t, err)
}

func TestExtract_EmptyHTML(t *testing.T) {
	// Empty HTML should not panic — readability may return an error or empty result
	result, err := readability.Extract("https://example.com/empty", "")
	// We accept either an error or an empty/sparse result
	if err == nil {
		assert.NotNil(t, result)
	}
}

func TestExtract_MinimalHTML(t *testing.T) {
	html := `<html><body><p>Hello world</p></body></html>`
	result, err := readability.Extract("https://example.com/minimal", html)
	// Should succeed without panic regardless of readability output quality
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestExtract_ResultFields(t *testing.T) {
	result, err := readability.Extract("https://example.com/go-modules", minimalArticleHTML)
	require.NoError(t, err)

	// Verify no leading/trailing whitespace on string fields
	assert.Equal(t, strings.TrimSpace(result.Title), result.Title)
	assert.Equal(t, strings.TrimSpace(result.Author), result.Author)
	assert.Equal(t, strings.TrimSpace(result.Excerpt), result.Excerpt)
	assert.Equal(t, strings.TrimSpace(result.Image), result.Image)
}
