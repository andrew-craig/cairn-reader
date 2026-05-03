// Package hash provides the canonical content-hash function for Cairn RSS
// ingestion. Both the Read and Explore services must use ContentHash so that
// deduplication keys are consistent across the pipeline.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ContentHash returns the lowercase hex SHA-256 of html after trimming leading
// and trailing whitespace. The trim step normalises minor whitespace differences
// that arise when the same article is fetched at different times.
func ContentHash(html []byte) string {
	normalized := strings.TrimSpace(string(html))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
