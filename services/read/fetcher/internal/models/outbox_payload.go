package models

// Outbox payload keys. Both the producer (processor.ItemProcessor) and the
// consumer (worker.OutboxWorker) reference these so the JSONB schema has a
// single source of truth.
const (
	PayloadKeyTitle          = "title"
	PayloadKeyAuthor         = "author"
	PayloadKeyPublishedAt    = "published_at"
	PayloadKeySourceURL      = "source_url"
	PayloadKeySourceFeedID   = "source_feed_id"
	PayloadKeyRawHTML        = "raw_html"
	PayloadKeyRawDescription = "raw_description"
)
