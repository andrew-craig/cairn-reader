import { getDb } from './db';
import { ReadService } from './read';
import { HttpError, UpdateUserContentRequest } from '@cairn/shared';

/**
 * The server's PATCH field names, plus `delete` for the archive (DELETE)
 * path. Deliberately the server's vocabulary, not the store's: the store's
 * scroll column is `scroll_fraction`, and `articles.scroll_position` is a
 * separate legacy column — see articleStore.ts. Payloads enqueued under
 * `status`/`is_favorite`/`scroll_position` are `UpdateUserContentRequest`
 * fragments and are replayed as-is; `delete` carries no payload.
 */
export type OutboxField = 'status' | 'is_favorite' | 'scroll_position' | 'delete';

interface OutboxRow {
  article_id: string;
  field: OutboxField;
  payload: string;
  created_at: number;
  attempts: number;
}

const ENQUEUE_SQL = `
  INSERT INTO outbox (article_id, field, payload, created_at, attempts)
  VALUES ($article_id, $field, $payload, $created_at, 0)
  ON CONFLICT(article_id, field) DO UPDATE SET payload = excluded.payload
`;

type SendOutcome = 'success' | 'drop' | 'halt';

/**
 * Replays one row against the backend and classifies the result:
 * - success: 2xx, or a 404 on a replayed `delete` (the server already has no
 *   record of it — exactly what the delete wanted).
 * - drop: a definitive 4xx other than 401. Replaying it again can only fail
 *   the same way.
 * - halt: NetworkError, 401, 5xx, or anything unrecognized. `fetchWithAuth`
 *   already retries once on 401 internally, so a 401 reaching here means
 *   refresh genuinely failed — transient from this module's point of view,
 *   same as a 5xx or an unreachable server.
 */
async function sendRow(row: OutboxRow): Promise<SendOutcome> {
  try {
    if (row.field === 'delete') {
      await ReadService.deleteUserContent(row.article_id);
    } else {
      await ReadService.updateUserContent(
        row.article_id,
        JSON.parse(row.payload) as UpdateUserContentRequest,
      );
    }
    return 'success';
  } catch (error) {
    if (row.field === 'delete' && error instanceof HttpError && error.status === 404) {
      return 'success';
    }
    if (error instanceof HttpError && error.status !== 401 && error.status < 500) {
      console.error(
        `Outbox: dropping ${row.field} write for article ${row.article_id} (non-retryable):`,
        error,
      );
      return 'drop';
    }
    // NetworkError, HttpError(401/5xx), or anything else unrecognized.
    return 'halt';
  }
}

export const Outbox = {
  /**
   * Queue a mutation for replay once connectivity returns. Coalesced by
   * (article_id, field): a later enqueue for the same pair replaces the
   * payload in place, leaving `created_at` untouched so the row keeps its
   * spot in the queue instead of jumping to the back. Enqueuing a `delete`
   * first clears any other pending row for that article — a PATCH replayed
   * against a since-deleted article is a guaranteed 404.
   */
  async enqueue(
    articleId: string,
    field: OutboxField,
    payload: Record<string, unknown>,
  ): Promise<void> {
    const db = await getDb();
    if (field === 'delete') {
      await db.runAsync("DELETE FROM outbox WHERE article_id = $article_id AND field != 'delete'", {
        $article_id: articleId,
      });
    }
    await db.runAsync(ENQUEUE_SQL, {
      $article_id: articleId,
      $field: field,
      $payload: JSON.stringify(payload),
      $created_at: Date.now(),
    });
  },

  /**
   * Replays queued mutations in FIFO order (`created_at`, then `rowid` to
   * break ties within the same millisecond — see task_ebf1 decision C).
   * Stops at the first row that comes back `halt` so a later row can never
   * reach the server ahead of an earlier one still pending: that row's
   * `attempts` is bumped (bookkeeping only, no backoff or drop-after-N in
   * v1) and the rest of the batch is left queued for the next drain.
   */
  async drain(): Promise<void> {
    const db = await getDb();
    const rows = await db.getAllAsync<OutboxRow>(
      'SELECT article_id, field, payload, created_at, attempts FROM outbox ORDER BY created_at ASC, rowid ASC',
    );
    for (const row of rows) {
      const outcome = await sendRow(row);
      if (outcome === 'halt') {
        await db.runAsync(
          'UPDATE outbox SET attempts = attempts + 1 WHERE article_id = $article_id AND field = $field',
          { $article_id: row.article_id, $field: row.field },
        );
        return;
      }
      await db.runAsync('DELETE FROM outbox WHERE article_id = $article_id AND field = $field', {
        $article_id: row.article_id,
        $field: row.field,
      });
    }
  },
};
