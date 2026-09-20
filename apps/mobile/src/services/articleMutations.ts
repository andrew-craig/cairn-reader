import { ArticleStore } from './articleStore';
import { ReadService } from './read';
import { Outbox, OutboxField, isRetryable } from './outbox';

/**
 * Runs a backend write; an error `isRetryable` (NetworkError, HttpError(401),
 * HttpError(5xx)) is queued for later replay instead of surfacing —
 * store-first, so the local write already happened and would otherwise
 * silently revert on the next sync with nothing in the outbox to freeze it
 * (task_c894). A definitive 4xx other than 401 is a real rejection and is
 * rethrown, exactly as an unqueued write would surface today.
 *
 * Uses the same `isRetryable` predicate as `outbox.ts`'s `sendRow` rather
 * than a private copy — a live 5xx and a replayed 5xx must be classified the
 * same way (see `outbox.ts` for why a live 401 in practice never reaches
 * here as an `HttpError`).
 */
async function withOutboxOnRetryableError(
  articleId: string,
  field: OutboxField,
  payload: Record<string, unknown>,
  send: () => Promise<unknown>,
): Promise<void> {
  try {
    await send();
  } catch (error) {
    if (isRetryable(error)) {
      await Outbox.enqueue(articleId, field, payload);
      return;
    }
    throw error;
  }
}

/**
 * Facade for the reading screen's mutations (task_ebf1): each writes the
 * local store first, then attempts the backend write, queuing it in the
 * outbox on a retryable error rather than losing it. Add-URL and Explore are
 * untouched — they stay online-only.
 */
export const ArticleMutations = {
  async markCompleted(articleId: string, readAt: number): Promise<void> {
    await ArticleStore.updateUserState(articleId, { isRead: true, readAt });
    await withOutboxOnRetryableError(articleId, 'status', { status: 'completed' }, () =>
      ReadService.updateUserContent(articleId, { status: 'completed' }),
    );
  },

  // No store write: the store tracks `is_read` (a boolean), not an
  // intermediate "reading" status, so there is nothing local to persist.
  async markReading(articleId: string): Promise<void> {
    await withOutboxOnRetryableError(articleId, 'status', { status: 'reading' }, () =>
      ReadService.updateUserContent(articleId, { status: 'reading' }),
    );
  },

  async saveScrollPosition(articleId: string, fraction: number): Promise<void> {
    await ArticleStore.updateUserState(articleId, { scrollFraction: fraction });
    await withOutboxOnRetryableError(
      articleId,
      'scroll_position',
      { scroll_position: fraction },
      () => ReadService.updateUserContent(articleId, { scroll_position: fraction }),
    );
  },

  async setFavorite(articleId: string, isFavorite: boolean): Promise<void> {
    await ArticleStore.updateUserState(articleId, { isFavorite });
    await withOutboxOnRetryableError(articleId, 'is_favorite', { is_favorite: isFavorite }, () =>
      ReadService.updateUserContent(articleId, { is_favorite: isFavorite }),
    );
  },

  async archive(articleId: string): Promise<void> {
    await ArticleStore.remove(articleId);
    await withOutboxOnRetryableError(articleId, 'delete', {}, () =>
      ReadService.deleteUserContent(articleId),
    );
  },
};
