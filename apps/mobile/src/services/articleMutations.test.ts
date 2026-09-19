import { ArticleMutations } from './articleMutations';
import { ArticleStore } from './articleStore';
import { ReadService } from './read';
import { Outbox } from './outbox';
import { HttpError, NetworkError } from '@cairn/shared';

// task_ebf1 (decision 4): each of the reading screen's six mutation call
// sites goes through this facade instead of repeating "write the store, try
// the network, enqueue on a retryable error" inline. task_c894: a retryable
// error (NetworkError, HttpError(401), HttpError(5xx) — the same set
// outbox.ts's sendRow treats as transient) enqueues; a definitive 4xx other
// than 401 surfaces exactly as it does without the facade.

jest.mock('./articleStore', () => ({
  ArticleStore: {
    updateUserState: jest.fn(),
    remove: jest.fn(),
  },
}));

jest.mock('./read', () => ({
  ReadService: {
    updateUserContent: jest.fn(),
    deleteUserContent: jest.fn(),
  },
}));

jest.mock('./outbox', () => ({
  ...jest.requireActual('./outbox'),
  Outbox: {
    enqueue: jest.fn(),
  },
}));

const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;
const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedOutbox = Outbox as jest.Mocked<typeof Outbox>;

describe('ArticleMutations', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedArticleStore.updateUserState.mockResolvedValue(undefined);
    mockedArticleStore.remove.mockResolvedValue(undefined);
    mockedOutbox.enqueue.mockResolvedValue(undefined);
  });

  describe('markCompleted', () => {
    it('writes the store, then the backend, in that order', async () => {
      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);

      await ArticleMutations.markCompleted('a1', 500);

      expect(mockedArticleStore.updateUserState).toHaveBeenCalledWith('a1', { isRead: true, readAt: 500 });
      expect(mockedReadService.updateUserContent).toHaveBeenCalledWith('a1', { status: 'completed' });
      const storeOrder = mockedArticleStore.updateUserState.mock.invocationCallOrder[0];
      const networkOrder = mockedReadService.updateUserContent.mock.invocationCallOrder[0];
      expect(storeOrder).toBeLessThan(networkOrder);
    });

    it('enqueues on NetworkError instead of throwing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new NetworkError());

      await expect(ArticleMutations.markCompleted('a1', 500)).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'status', { status: 'completed' });
    });

    it('surfaces a definitive 4xx instead of enqueuing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new HttpError(422, 'nope'));

      await expect(ArticleMutations.markCompleted('a1', 500)).rejects.toThrow('nope');

      expect(mockedOutbox.enqueue).not.toHaveBeenCalled();
    });

    // task_c894: a live 503 was previously indistinguishable from a
    // definitive 4xx here, so the DELETE/PATCH was rethrown after the store
    // write had already landed, and nothing was queued to freeze the value
    // against the next sync.
    it('enqueues on a live HttpError(503) instead of throwing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new HttpError(503, 'Service Unavailable'));

      await expect(ArticleMutations.markCompleted('a1', 500)).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'status', { status: 'completed' });
    });
  });

  describe('markReading', () => {
    it('does not touch the store (no store field for "reading")', async () => {
      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);

      await ArticleMutations.markReading('a1');

      expect(mockedArticleStore.updateUserState).not.toHaveBeenCalled();
      expect(mockedReadService.updateUserContent).toHaveBeenCalledWith('a1', { status: 'reading' });
    });

    it('enqueues on NetworkError', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new NetworkError());

      await expect(ArticleMutations.markReading('a1')).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'status', { status: 'reading' });
    });

    it('enqueues on a live HttpError(503) instead of throwing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new HttpError(503, 'Service Unavailable'));

      await expect(ArticleMutations.markReading('a1')).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'status', { status: 'reading' });
    });
  });

  describe('saveScrollPosition', () => {
    it('writes the store, then the backend', async () => {
      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);

      await ArticleMutations.saveScrollPosition('a1', 0.42);

      expect(mockedArticleStore.updateUserState).toHaveBeenCalledWith('a1', { scrollFraction: 0.42 });
      expect(mockedReadService.updateUserContent).toHaveBeenCalledWith('a1', { scroll_position: 0.42 });
    });

    it('enqueues on NetworkError under the scroll_position field', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new NetworkError());

      await ArticleMutations.saveScrollPosition('a1', 0.42);

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'scroll_position', { scroll_position: 0.42 });
    });

    it('enqueues on a live HttpError(503) instead of throwing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new HttpError(503, 'Service Unavailable'));

      await expect(ArticleMutations.saveScrollPosition('a1', 0.42)).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'scroll_position', { scroll_position: 0.42 });
    });
  });

  describe('setFavorite', () => {
    it('writes the store, then the backend', async () => {
      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);

      await ArticleMutations.setFavorite('a1', true);

      expect(mockedArticleStore.updateUserState).toHaveBeenCalledWith('a1', { isFavorite: true });
      expect(mockedReadService.updateUserContent).toHaveBeenCalledWith('a1', { is_favorite: true });
    });

    it('enqueues on NetworkError', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new NetworkError());

      await expect(ArticleMutations.setFavorite('a1', true)).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'is_favorite', { is_favorite: true });
    });

    it('surfaces a definitive 4xx instead of enqueuing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new HttpError(400, 'bad'));

      await expect(ArticleMutations.setFavorite('a1', true)).rejects.toThrow('bad');

      expect(mockedOutbox.enqueue).not.toHaveBeenCalled();
    });

    it('enqueues on a live HttpError(503) instead of throwing', async () => {
      mockedReadService.updateUserContent.mockRejectedValue(new HttpError(503, 'Service Unavailable'));

      await expect(ArticleMutations.setFavorite('a1', true)).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'is_favorite', { is_favorite: true });
    });
  });

  describe('archive', () => {
    it('removes from the store, then deletes on the backend', async () => {
      mockedReadService.deleteUserContent.mockResolvedValue(undefined);

      await ArticleMutations.archive('a1');

      expect(mockedArticleStore.remove).toHaveBeenCalledWith('a1');
      expect(mockedReadService.deleteUserContent).toHaveBeenCalledWith('a1');
      const storeOrder = mockedArticleStore.remove.mock.invocationCallOrder[0];
      const networkOrder = mockedReadService.deleteUserContent.mock.invocationCallOrder[0];
      expect(storeOrder).toBeLessThan(networkOrder);
    });

    it('enqueues a delete on NetworkError instead of throwing', async () => {
      mockedReadService.deleteUserContent.mockRejectedValue(new NetworkError());

      await expect(ArticleMutations.archive('a1')).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'delete', {});
    });

    it('surfaces a definitive 4xx instead of enqueuing (fixes the swallowed archive error)', async () => {
      mockedReadService.deleteUserContent.mockRejectedValue(new HttpError(403, 'forbidden'));

      await expect(ArticleMutations.archive('a1')).rejects.toThrow('forbidden');

      expect(mockedOutbox.enqueue).not.toHaveBeenCalled();
    });

    // The motivating case from task_c894: online, backend returns 503 on
    // archive — ArticleStore.remove() already ran, and without this the
    // DELETE would be rethrown with nothing queued, leaving the article gone
    // locally but still present on the server.
    it('enqueues a delete on a live HttpError(503) instead of throwing', async () => {
      mockedReadService.deleteUserContent.mockRejectedValue(new HttpError(503, 'Service Unavailable'));

      await expect(ArticleMutations.archive('a1')).resolves.toBeUndefined();

      expect(mockedOutbox.enqueue).toHaveBeenCalledWith('a1', 'delete', {});
    });
  });
});
