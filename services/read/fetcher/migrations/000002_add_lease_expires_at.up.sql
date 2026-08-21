-- Add lease-based atomic claim support to feed_items, content_outbox, and feeds.
--
-- These are the rows workers pick up ("pending"/"processing", "pending"/"sending",
-- or a feed due for polling) and process. Without a lease, a crashed worker leaves
-- a row stuck in its in-flight status forever (no query ever re-selects it), and
-- widening the selector to include the in-flight status without an atomic claim
-- lets two workers pick up the same row concurrently. lease_expires_at supports a
-- short-lived claim (an atomic SELECT ... FOR UPDATE SKIP LOCKED + UPDATE) that is
-- held only for the instant of the claim, not for the duration of the underlying
-- HTTP work, and expires on its own if the worker that claimed the row dies.

ALTER TABLE feed_items ADD COLUMN lease_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE content_outbox ADD COLUMN lease_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE feeds ADD COLUMN lease_expires_at TIMESTAMP WITH TIME ZONE;

-- idx_feed_items_status (processing_status, discovered_at) already covers the
-- 'pending' branch of the widened selector efficiently. Add a partial index for
-- the 'processing' branch so an expired-lease scan doesn't fall back to a full
-- table scan once processing_status is no longer a leading, low-cardinality filter
-- shared with 'pending'.
CREATE INDEX idx_feed_items_processing_lease ON feed_items(lease_expires_at)
    WHERE processing_status = 'processing';

-- idx_outbox_pending (next_retry_at) WHERE delivery_status IN ('pending', 'sending')
-- already covers content_outbox's widened selector (delivery_status IN (...) AND
-- next_retry_at <= now()); no additional index needed for the lease branch since
-- next_retry_at <= now() is not gated on the lease column.

-- feeds' existing idx_feeds_next_poll (next_poll_at) WHERE status = 'active' already
-- covers the widened predicate (status = 'active' AND next_poll_at <= now() AND
-- (lease_expires_at IS NULL OR lease_expires_at < now())); the lease check is an
-- extra filter evaluated against the small set of rows the next_poll_at index
-- narrows down to, not a second index range scan.
