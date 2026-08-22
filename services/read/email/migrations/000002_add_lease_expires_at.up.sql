-- Add lease-based atomic claim support to raw_emails and content_outbox.
--
-- These are the rows workers pick up ("pending"/"processing", "pending"/"sending")
-- and process. Without a lease, a crashed worker leaves a row stuck in its
-- in-flight status forever (no query ever re-selects it), and widening the
-- selector to include the in-flight status without an atomic claim lets two
-- workers pick up the same row concurrently. lease_expires_at supports a
-- short-lived claim (an atomic SELECT ... FOR UPDATE SKIP LOCKED + UPDATE) that is
-- held only for the instant of the claim, not for the duration of the underlying
-- HTTP work, and expires on its own if the worker that claimed the row dies.

ALTER TABLE raw_emails ADD COLUMN lease_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE content_outbox ADD COLUMN lease_expires_at TIMESTAMP WITH TIME ZONE;

-- idx_raw_emails_status (processing_status, created_at) WHERE processing_status
-- IN ('pending', 'processing') already anticipated this widened selector's status
-- branch. Adding retry_count < 5 as an extra predicate doesn't invalidate that
-- index -- it's applied as a filter on the rows the index already narrows down to.
-- No change needed here.

-- idx_content_outbox_pending (next_retry_at) WHERE delivery_status IN ('pending',
-- 'sending') already covers content_outbox's widened selector; no additional
-- index needed for the lease branch since next_retry_at <= now() is not gated on
-- the lease column.
