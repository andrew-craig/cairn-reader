import { Outbox } from './outbox';
import { ArticleStore } from './articleStore';
import { ReadService } from './read';
import { HttpError, NetworkError } from '../utils/errors';
import { getDb } from './db';

// __mocks__/expo-sqlite.js adapts openDatabaseAsync onto a real node:sqlite
// DatabaseSync(':memory:'), so these tests exercise the real schema and real
// SQL — see articleStore.test.ts for the same rationale. ArticleStore.clear()
// clears both tables sharing that database and is reused here for isolation.
jest.mock('expo-sqlite');

jest.mock('./read', () => ({
  ReadService: {
    updateUserContent: jest.fn(),
    deleteUserContent: jest.fn(),
  },
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;

describe('Outbox', () => {
  beforeEach(async () => {
    jest.clearAllMocks();
    await ArticleStore.clear();
  });

  describe('enqueue', () => {
    it('coalesces repeated writes to the same (article_id, field) and preserves created_at', async () => {
      await Outbox.enqueue('a1', 'scroll_position', { scroll_position: 0.1 });
      const db = await getDb();
      const firstRow = await db.getFirstAsync<{ created_at: number }>(
        'SELECT created_at FROM outbox WHERE article_id = $a AND field = $f',
        { $a: 'a1', $f: 'scroll_position' },
      );

      await Outbox.enqueue('a1', 'scroll_position', { scroll_position: 0.9 });

      const rows = await db.getAllAsync<{ payload: string; created_at: number }>(
        'SELECT payload, created_at FROM outbox WHERE article_id = $a AND field = $f',
        { $a: 'a1', $f: 'scroll_position' },
      );
      expect(rows).toHaveLength(1);
      expect(JSON.parse(rows[0].payload)).toEqual({ scroll_position: 0.9 });
      expect(rows[0].created_at).toBe(firstRow?.created_at);
    });

    it('keeps separate rows for different fields on the same article', async () => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      await Outbox.enqueue('a1', 'scroll_position', { scroll_position: 0.5 });

      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);
      await Outbox.drain();

      expect(mockedReadService.updateUserContent).toHaveBeenCalledTimes(2);
    });

    it('enqueuing a delete supersedes that article\'s other pending rows', async () => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      await Outbox.enqueue('a1', 'scroll_position', { scroll_position: 0.5 });

      await Outbox.enqueue('a1', 'delete', {});

      mockedReadService.deleteUserContent.mockResolvedValue(undefined);
      await Outbox.drain();

      // Only the delete should have been replayed — the superseded PATCH
      // rows must never reach the network (a 404 waiting to happen).
      expect(mockedReadService.updateUserContent).not.toHaveBeenCalled();
      expect(mockedReadService.deleteUserContent).toHaveBeenCalledTimes(1);
      expect(mockedReadService.deleteUserContent).toHaveBeenCalledWith('a1');
    });

    it('does not disturb a different article\'s pending rows', async () => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      await Outbox.enqueue('a2', 'delete', {});

      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);
      mockedReadService.deleteUserContent.mockResolvedValue(undefined);
      await Outbox.drain();

      expect(mockedReadService.updateUserContent).toHaveBeenCalledWith('a1', { is_favorite: true });
      expect(mockedReadService.deleteUserContent).toHaveBeenCalledWith('a2');
    });
  });

  describe('drain: ordering', () => {
    it('replays rows in created_at order', async () => {
      const db = await getDb();
      // Seed directly so created_at is fully controlled (enqueue always uses
      // Date.now(), which can't be relied on to differ across fast calls).
      await db.runAsync(
        "INSERT INTO outbox (article_id, field, payload, created_at, attempts) VALUES ('a2', 'is_favorite', '{\"is_favorite\":true}', 200, 0)",
      );
      await db.runAsync(
        "INSERT INTO outbox (article_id, field, payload, created_at, attempts) VALUES ('a1', 'is_favorite', '{\"is_favorite\":true}', 100, 0)",
      );

      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);
      await Outbox.drain();

      expect(mockedReadService.updateUserContent.mock.calls[0][0]).toBe('a1');
      expect(mockedReadService.updateUserContent.mock.calls[1][0]).toBe('a2');
    });

    it('breaks a created_at tie with insertion order (rowid)', async () => {
      const db = await getDb();
      // Same millisecond timestamp for both — only insertion order (rowid)
      // can decide replay order.
      await db.runAsync(
        "INSERT INTO outbox (article_id, field, payload, created_at, attempts) VALUES ('first', 'is_favorite', '{\"is_favorite\":true}', 500, 0)",
      );
      await db.runAsync(
        "INSERT INTO outbox (article_id, field, payload, created_at, attempts) VALUES ('second', 'is_favorite', '{\"is_favorite\":true}', 500, 0)",
      );

      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);
      await Outbox.drain();

      expect(mockedReadService.updateUserContent.mock.calls[0][0]).toBe('first');
      expect(mockedReadService.updateUserContent.mock.calls[1][0]).toBe('second');
    });
  });

  describe('drain: outcomes', () => {
    it('a 2xx (resolved) write deletes the row', async () => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      mockedReadService.updateUserContent.mockResolvedValue(undefined as never);

      await Outbox.drain();

      const db = await getDb();
      const remaining = await db.getAllAsync('SELECT * FROM outbox');
      expect(remaining).toHaveLength(0);
    });

    it('a 404 on a replayed delete counts as success and deletes the row', async () => {
      await Outbox.enqueue('a1', 'delete', {});
      mockedReadService.deleteUserContent.mockRejectedValue(new HttpError(404, 'Not found'));

      await Outbox.drain();

      const db = await getDb();
      const remaining = await db.getAllAsync('SELECT * FROM outbox');
      expect(remaining).toHaveLength(0);
    });

    it('a definitive 4xx (not 401) drops the row and continues the batch', async () => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      await Outbox.enqueue('a2', 'is_favorite', { is_favorite: true });
      mockedReadService.updateUserContent
        .mockRejectedValueOnce(new HttpError(422, 'Unprocessable'))
        .mockResolvedValueOnce(undefined as never);
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      await Outbox.drain();

      const db = await getDb();
      const remaining = await db.getAllAsync('SELECT * FROM outbox');
      expect(remaining).toHaveLength(0);
      expect(mockedReadService.updateUserContent).toHaveBeenCalledTimes(2);
      expect(consoleErrorSpy).toHaveBeenCalled();
      consoleErrorSpy.mockRestore();
    });

    it.each([
      ['NetworkError', () => new NetworkError()],
      ['HttpError 401', () => new HttpError(401, 'Unauthorized')],
      ['HttpError 500', () => new HttpError(500, 'Server error')],
    ])('%s keeps the row, bumps attempts, and halts the rest of the batch', async (_label, makeError) => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      await Outbox.enqueue('a2', 'is_favorite', { is_favorite: true });
      mockedReadService.updateUserContent.mockRejectedValue(makeError());

      await Outbox.drain();

      expect(mockedReadService.updateUserContent).toHaveBeenCalledTimes(1);
      expect(mockedReadService.updateUserContent).toHaveBeenCalledWith('a1', { is_favorite: true });

      const db = await getDb();
      const remaining = await db.getAllAsync<{ article_id: string; attempts: number }>(
        'SELECT article_id, attempts FROM outbox ORDER BY article_id',
      );
      expect(remaining.map((r) => r.article_id)).toEqual(['a1', 'a2']);
      expect(remaining.find((r) => r.article_id === 'a1')?.attempts).toBe(1);
      expect(remaining.find((r) => r.article_id === 'a2')?.attempts).toBe(0);
    });

    it('a second drain retries a halted row from where it left off', async () => {
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });
      mockedReadService.updateUserContent.mockRejectedValueOnce(new NetworkError());

      await Outbox.drain();
      let db = await getDb();
      let row = await db.getFirstAsync<{ attempts: number }>(
        "SELECT attempts FROM outbox WHERE article_id = 'a1'",
      );
      expect(row?.attempts).toBe(1);

      mockedReadService.updateUserContent.mockResolvedValueOnce(undefined as never);
      await Outbox.drain();

      db = await getDb();
      const remaining = await db.getAllAsync('SELECT * FROM outbox');
      expect(remaining).toHaveLength(0);
    });
  });
});
