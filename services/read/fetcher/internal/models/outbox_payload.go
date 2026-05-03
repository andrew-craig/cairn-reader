package models

// Outbox payload keys. Both the producer (processor.ItemProcessor) and the
// consumer (worker.OutboxWorker) reference these so the JSONB schema has a
// single source of truth.
//
// PayloadKeyCleanedHTML is the legacy key written by the pre-task_7479
// fetcher (when readability + sanitize ran in the fetcher). It is read on the
// consumer side for one release cycle to drain in-flight outbox entries, and
// can be removed once the queue is fully drained.
const (
	PayloadKeyTitle          = "title"
	PayloadKeyAuthor         = "author"
	PayloadKeyPublishedAt    = "published_at"
	PayloadKeySourceURL      = "source_url"
	PayloadKeySourceFeedID   = "source_feed_id"
	PayloadKeyRawHTML        = "raw_html"
	PayloadKeyRawDescription = "raw_description"

	PayloadKeyCleanedHTML = "cleaned_html"
)
