import { ArticleStore } from './articleStore';
import { ReadService } from './read';
import { isOffline } from '../utils/network';
import { NetworkError } from '@cairn/shared';

// Cap on both the prefetch selection and the post-run eviction — the same
// "100 most recent Read-list articles" window the task describes.
const PREFETCH_LIMIT = 100;
// Small fixed worker pool. No new dependency (no p-limit or similar).
const CONCURRENCY = 3;

// A run already in progress must not be restarted by a second sync (e.g.
// pull-to-refresh while focus-refresh prefetch is still draining). This is a
// no-op guard, not a queue: a second call while one is running does nothing.
let inFlight = false;

/**
 * Downloads cleaned_html for unread/reading Read-list articles that lack a
 * body, bounded concurrency, newest first, capped at PREFETCH_LIMIT. Called
 * from ReadScreen's sync callback after ArticleStore.upsertMany — see
 * apps/mobile/CLAUDE.md and task_c55c. Explore article content is never
 * synced or prefetched.
 */
export const ArticlePrefetchService = {
  async run(): Promise<void> {
    if (inFlight) return;
    inFlight = true;
    try {
      // Never prefetch while offline — isOffline() is the same one-off check
      // used by non-React service code elsewhere in the app.
      if (await isOffline()) return;

      const candidates = await ArticleStore.listPrefetchCandidates(PREFETCH_LIMIT);
      if (candidates.length > 0) {
        await runPool(candidates.map((article) => article.id));
      }

      // Enforce the retention cap regardless of how the batch above ended —
      // eviction is bookkeeping, not part of the network pass it just ran.
      await ArticleStore.evictBodiesOutsideCap(PREFETCH_LIMIT);
    } finally {
      inFlight = false;
    }
  },
};

/**
 * Fetches and stores each candidate's body with CONCURRENCY workers pulling
 * from a shared queue. A NetworkError means the connection went away — abort
 * the remaining queue rather than burning the rest of the batch on failing
 * requests. Any other per-article error is logged and skipped; the batch
 * continues.
 */
async function runPool(ids: string[]): Promise<void> {
  let nextIndex = 0;
  let aborted = false;

  async function worker(): Promise<void> {
    while (!aborted) {
      const index = nextIndex++;
      if (index >= ids.length) return;
      const id = ids[index];
      try {
        const detail = await ReadService.getContentById(id);
        const { content } = ReadService.transformDetailToArticle(detail);
        if (content) {
          await ArticleStore.saveBody(id, content);
        }
      } catch (error) {
        if (error instanceof NetworkError) {
          aborted = true;
          return;
        }
        console.error('Failed to prefetch article body:', id, error);
      }
    }
  }

  const workerCount = Math.min(CONCURRENCY, ids.length);
  await Promise.all(Array.from({ length: workerCount }, () => worker()));
}
