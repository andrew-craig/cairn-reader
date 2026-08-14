package handlers

// Request body size limits, mirroring the pattern in
// explore/recommender/internal/api/handlers.go. Content-carrying endpoints
// get a generous cap; simple JSON endpoints (URL strings, IDs, statuses)
// get a tight one.
const (
	maxContentBodySize     = 10 << 20 // 10MB: single content create/update (HTML up to 5MB + overhead)
	maxBulkContentBodySize = 50 << 20 // 50MB: bulk content create (up to 100 items, 5MB each max)
	maxSimpleRequestSize   = 16 << 10 // 16KB: single URL/status/id-only request bodies
	maxSmallBatchBodySize  = 64 << 10 // 64KB: up to 100 items of ids/hashes/statuses (no HTML)
)
