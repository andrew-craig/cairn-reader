import { ArticleStore } from './articleStore';
import { ReadService } from './read';
import { Outbox, OutboxField } from './outbox';
import { NetworkError } from '../utils/errors';

/**
 * Runs a backend write; a `NetworkError` is queued for later replay instead
 * of surfacing (store-first — the local write already happened). Any other
 * rejection (a definitive 4xx, an auth failure, ...) is a real rejection and
 * is rethrown, exactly as an unqueued write would surface today.
 */
async function withOutboxOnNetworkError(
  articleId: string,
  field: OutboxField,
  payload: Record<string, unknown>,
  send: () => Promise<unknown>,
): Promise<void> {
  try {
    await send();
  } catch (error) {
    if (error instanceof NetworkError) {
      await Outbox.enqueue(articleId, field, payload);
      return;
    }
    throw error;
  }
}

/**
 * Facade for the reading screen's mutations (task_ebf1): each writes the
 * local store first, then attempts the backend write, queuing it in the
 * outbox on `NetworkError` rather than losing it. Add-URL and Explore are
 * untouched — they stay online-only.
 */
export const ArticleMutations = {
  async markCompleted(articleId: string, readAt: number): Promise<void> {
    await ArticleStore.updateUserState(articleId, { isRead: true, readAt });
    await withOutboxOnNetworkError(articleId, 'status', { status: 'completed' }, () =>
      ReadService.updateUserContent(articleId, { status: 'completed' }),
    );
  },

  // No store write: the store tracks `is_read` (a boolean), not an
  // intermediate "reading" status, so there is nothing local to persist.
  async markReading(articleId: string): Promise<void> {
    await withOutboxOnNetworkError(articleId, 'status', { status: 'reading' }, () =>
      ReadService.updateUserContent(articleId, { status: 'reading' }),
    );
  },

  async saveScrollPosition(articleId: string, fraction: number): Promise<void> {
    await ArticleStore.updateUserState(articleId, { scrollFraction: fraction });
    await withOutboxOnNetworkError(
      articleId,
      'scroll_position',
      { scroll_position: fraction },
      () => ReadService.updateUserContent(articleId, { scroll_position: fraction }),
    );
  },

  async setFavorite(articleId: string, isFavorite: boolean): Promise<void> {
    await ArticleStore.updateUserState(articleId, { isFavorite });
    await withOutboxOnNetworkError(articleId, 'is_favorite', { is_favorite: isFavorite }, () =>
      ReadService.updateUserContent(articleId, { is_favorite: isFavorite }),
    );
  },

  async archive(articleId: string): Promise<void> {
    await ArticleStore.remove(articleId);
    await withOutboxOnNetworkError(articleId, 'delete', {}, () =>
      ReadService.deleteUserContent(articleId),
    );
  },
};
