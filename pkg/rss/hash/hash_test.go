package hash_test

import (
	"strings"
	"testing"

	"github.com/cairn-app/cairn-reader/pkg/rss/hash"
	"github.com/stretchr/testify/assert"
)

func TestContentHash_Format(t *testing.T) {
	h := hash.ContentHash([]byte("<p>hello</p>"))
	// SHA-256 hex is always 64 lowercase characters
	assert.Len(t, h, 64)
	assert.Equal(t, strings.ToLower(h), h)
}

func TestContentHash_Deterministic(t *testing.T) {
	html := []byte("<article><p>Same content</p></article>")
	assert.Equal(t, hash.ContentHash(html), hash.ContentHash(html))
}

func TestContentHash_DifferentInputsDifferentHash(t *testing.T) {
	a := hash.ContentHash([]byte("<p>content A</p>"))
	b := hash.ContentHash([]byte("<p>content B</p>"))
	assert.NotEqual(t, a, b)
}

func TestContentHash_WhitespaceTrimmed(t *testing.T) {
	// Leading/trailing whitespace differences must not affect the hash
	base := hash.ContentHash([]byte("<p>hello</p>"))
	padded := hash.ContentHash([]byte("  <p>hello</p>  "))
	trailing := hash.ContentHash([]byte("<p>hello</p>\n"))
	assert.Equal(t, base, padded)
	assert.Equal(t, base, trailing)
}

func TestContentHash_EmptyInput(t *testing.T) {
	h := hash.ContentHash([]byte(""))
	// Must not panic; returns a valid 64-char hex string
	assert.Len(t, h, 64)
}

func TestContentHash_LargeInput(t *testing.T) {
	large := []byte(strings.Repeat("<p>x</p>", 100_000))
	h := hash.ContentHash(large)
	assert.Len(t, h, 64)
}
