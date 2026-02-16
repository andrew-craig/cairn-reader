// Package processor provides email content processing functionality.
package processor

// ContentExtractor handles readability extraction and HTML sanitization.
// Responsibilities:
// - Apply go-readability for content extraction
// - HTML sanitization using bluemonday
// - Generate content hash for deduplication
