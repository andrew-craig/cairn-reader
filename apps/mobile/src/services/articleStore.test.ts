import { ArticleStore } from './articleStore';
import { Outbox } from './outbox';
import { getDb } from './db';
import { Article } from '../types';

// __mocks__/expo-sqlite.js adapts openDatabaseAsync onto a real
// node:sqlite DatabaseSync(':memory:'), so these tests exercise the real
// schema and real SQL rather than a fake that can't disagree with them.
jest.mock('expo-sqlite');

const makeArticle = (overrides: Partial<Article> = {}): Article => ({
  id: 'a1',
  url: 'https://example.com/a1',
  title: 'Article One',
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: 1000,
  ...overrides,
});

describe('ArticleStore', () => {
  beforeEach(async () => {
    await ArticleStore.clear();
  });

  describe('upsertMany', () => {
    it('inserts new articles', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' }), makeArticle({ id: 'a2' })]);

      const recent = await ArticleStore.listRecent(10);
      expect(recent).toHaveLength(2);
    });

    it('updates an existing row instead of duplicating it', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', title: 'Old Title' })]);
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', title: 'New Title' })]);

      const recent = await ArticleStore.listRecent(10);
      expect(recent).toHaveLength(1);
      expect(recent[0].title).toBe('New Title');
    });

    it('does not delete rows absent from the incoming page (upsert-only sync)', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);
      await ArticleStore.upsertMany([makeArticle({ id: 'a2' })]);

      const recent = await ArticleStore.listRecent(10);
      expect(recent.map((a) => a.id).sort()).toEqual(['a1', 'a2']);
    });
  });

  describe('listRecent', () => {
    it('orders by addedAt descending and respects the limit', async () => {
      await ArticleStore.upsertMany([
        makeArticle({ id: 'old', addedAt: 100 }),
        makeArticle({ id: 'newest', addedAt: 300 }),
        makeArticle({ id: 'middle', addedAt: 200 }),
      ]);

      const recent = await ArticleStore.listRecent(2);
      expect(recent.map((a) => a.id)).toEqual(['newest', 'middle']);
    });
  });

  describe('listFavorites', () => {
    it('returns only articles marked as favorite', async () => {
      await ArticleStore.upsertMany([
        makeArticle({ id: 'fav', isFavorite: true, addedAt: 100 }),
        makeArticle({ id: 'not-fav', isFavorite: false, addedAt: 200 }),
      ]);

      const favorites = await ArticleStore.listFavorites();
      expect(favorites.map((a) => a.id)).toEqual(['fav']);
    });
  });

  describe('body round-tripping', () => {
    it('returns content undefined when no body has been saved', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.content).toBeUndefined();
    });

    it('round-trips a saved body through getById', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);
      await ArticleStore.saveBody('a1', '<p>Hello</p>');

      const stored = await ArticleStore.getById('a1');
      expect(stored?.content).toBe('<p>Hello</p>');
    });

    it('preserves an existing body when a later list sync has no content', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);
      await ArticleStore.saveBody('a1', '<p>Hello</p>');

      // A list-page refresh never carries cleaned_html (content undefined).
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', title: 'Refreshed' })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.title).toBe('Refreshed');
      expect(stored?.content).toBe('<p>Hello</p>');
    });
  });

  // task_c55c: upsertMany diffs content_hash so a body whose hash no longer
  // matches the server's is invalidated immediately, rather than only being
  // caught later by comparing against a fresh fetch. Pins both halves: a
  // refresh with the same hash keeps the body, a refresh with a changed hash
  // clears it, and a refresh with no hash at all keeps it (the pre-existing
  // COALESCE behaviour list pages rely on, since they never carry a hash on
  // the article record itself in this store's terms of "no content_hash").
  describe('content_hash body invalidation', () => {
    it('keeps the body when a refreshed row carries the same content_hash', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', contentHash: 'h1' })]);
      await ArticleStore.saveBody('a1', '<p>Hello</p>');

      await ArticleStore.upsertMany([
        makeArticle({ id: 'a1', title: 'Refreshed', contentHash: 'h1' }),
      ]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.title).toBe('Refreshed');
      expect(stored?.content).toBe('<p>Hello</p>');
      expect(stored?.contentHash).toBe('h1');
    });

    it('clears the body when a refreshed row carries a different content_hash', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', contentHash: 'h1' })]);
      await ArticleStore.saveBody('a1', '<p>Hello</p>');

      await ArticleStore.upsertMany([makeArticle({ id: 'a1', contentHash: 'h2' })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.content).toBeUndefined();
      expect(stored?.contentHash).toBe('h2');
    });

    it('keeps the body when a refreshed row carries no content_hash at all', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', contentHash: 'h1' })]);
      await ArticleStore.saveBody('a1', '<p>Hello</p>');

      await ArticleStore.upsertMany([makeArticle({ id: 'a1', contentHash: undefined })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.content).toBe('<p>Hello</p>');
      expect(stored?.contentHash).toBe('h1');
    });
  });

  describe('listPrefetchCandidates', () => {
    it('selects unread articles with no cached body, newest first', async () => {
      await ArticleStore.upsertMany([
        makeArticle({ id: 'no-body', addedAt: 100 }),
        makeArticle({ id: 'newer-no-body', addedAt: 300 }),
        makeArticle({ id: 'has-body', addedAt: 200 }),
        makeArticle({ id: 'read', addedAt: 400, isRead: true }),
      ]);
      await ArticleStore.saveBody('has-body', '<p>already cached</p>');

      const candidates = await ArticleStore.listPrefetchCandidates(10);

      expect(candidates.map((a) => a.id)).toEqual(['newer-no-body', 'no-body']);
    });

    it('respects the limit', async () => {
      await ArticleStore.upsertMany([
        makeArticle({ id: 'a', addedAt: 100 }),
        makeArticle({ id: 'b', addedAt: 200 }),
        makeArticle({ id: 'c', addedAt: 300 }),
      ]);

      const candidates = await ArticleStore.listPrefetchCandidates(2);

      expect(candidates.map((a) => a.id)).toEqual(['c', 'b']);
    });
  });

  describe('evictBodiesOutsideCap', () => {
    it('clears bodies for rows outside the cap, keeping the most recent', async () => {
      await ArticleStore.upsertMany([
        makeArticle({ id: 'old', addedAt: 100 }),
        makeArticle({ id: 'newer', addedAt: 200 }),
      ]);
      await ArticleStore.saveBody('old', '<p>old</p>');
      await ArticleStore.saveBody('newer', '<p>newer</p>');

      await ArticleStore.evictBodiesOutsideCap(1);

      const old = await ArticleStore.getById('old');
      const newer = await ArticleStore.getById('newer');
      expect(old?.content).toBeUndefined();
      expect(newer?.content).toBe('<p>newer</p>');
    });

    it('does not touch metadata, only body', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'old', addedAt: 100, title: 'Old' })]);
      await ArticleStore.saveBody('old', '<p>old</p>');

      await ArticleStore.evictBodiesOutsideCap(0);

      const old = await ArticleStore.getById('old');
      expect(old?.title).toBe('Old');
      expect(old?.content).toBeUndefined();
    });
  });

  describe('updateUserState', () => {
    it('updates isRead, isFavorite and scrollFraction independently', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);

      await ArticleStore.updateUserState('a1', { isFavorite: true });
      let stored = await ArticleStore.getById('a1');
      expect(stored?.isFavorite).toBe(true);
      expect(stored?.isRead).toBe(false);

      await ArticleStore.updateUserState('a1', { isRead: true, readAt: 500 });
      stored = await ArticleStore.getById('a1');
      expect(stored?.isRead).toBe(true);
      expect(stored?.readAt).toBe(500);
      expect(stored?.isFavorite).toBe(true);

      await ArticleStore.updateUserState('a1', { scrollFraction: 0.42 });
      stored = await ArticleStore.getById('a1');
      expect(stored?.scrollFraction).toBe(0.42);
    });

    it('does nothing when the id is not found', async () => {
      await expect(ArticleStore.updateUserState('missing', { isRead: true })).resolves.toBeUndefined();
    });
  });

  describe('remove', () => {
    it('deletes a single article by id', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' }), makeArticle({ id: 'a2' })]);

      await ArticleStore.remove('a1');

      const recent = await ArticleStore.listRecent(10);
      expect(recent.map((a) => a.id)).toEqual(['a2']);
    });
  });

  describe('clear', () => {
    it('removes all stored articles', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' }), makeArticle({ id: 'a2' })]);

      await ArticleStore.clear();

      const recent = await ArticleStore.listRecent(10);
      expect(recent).toEqual([]);
    });

    // task_ebf1 (module layout, item F): a queued write must never replay
    // against a different account — clear() (called on logout) drops the
    // outbox alongside the articles it shares a database with.
    it('also clears queued outbox rows', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', isFavorite: true })]);
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });

      await ArticleStore.clear();

      // Re-seed the row (a plain insert, articles is empty post-clear) and
      // then run a conflicting upsert. If clear() had left the outbox row
      // behind, upsertMany's guard would freeze isFavorite instead of
      // accepting the incoming value — so this only passes if the outbox
      // was actually cleared.
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', isFavorite: true })]);
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', isFavorite: false })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.isFavorite).toBe(false);
    });
  });

  // task_ebf1: the tech lead's note on task_a8a4's review — upsertMany's
  // "server rows win" sync would otherwise clobber a value the user just
  // changed offline while its write is still queued. See scope clarification
  // decision 1 (guard in SQL) and pre-assignment review item B (pending
  // delete).
  describe('outbox guard on upsertMany (interleaving with a pending write)', () => {
    it('preserves a pending offline change when a list sync arrives before the drain', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', isFavorite: false, title: 'Original' })]);
      // Simulates ArticleMutations.setFavorite: store write happens first,
      // the network write fails with a NetworkError and gets queued.
      await ArticleStore.updateUserState('a1', { isFavorite: true });
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });

      // A list sync arrives before the outbox drains, carrying the server's
      // still-stale value.
      await ArticleStore.upsertMany([
        makeArticle({ id: 'a1', isFavorite: false, title: 'Refreshed' }),
      ]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.isFavorite).toBe(true); // survives
      expect(stored?.title).toBe('Refreshed'); // non-user-state columns still sync normally
    });

    it('does not resurrect an article with a pending delete (item B)', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);
      await ArticleStore.remove('a1');
      await Outbox.enqueue('a1', 'delete', {});

      // The next list sync still sees the article server-side.
      await ArticleStore.upsertMany([makeArticle({ id: 'a1' })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored).toBeNull();
    });

    it('resumes normal syncing once the outbox row is gone', async () => {
      await ArticleStore.upsertMany([makeArticle({ id: 'a1', isFavorite: true })]);
      await Outbox.enqueue('a1', 'is_favorite', { is_favorite: true });

      // Drain succeeded and cleared its row — simulate directly.
      const db = await getDb();
      await db.runAsync("DELETE FROM outbox WHERE article_id = 'a1'");

      await ArticleStore.upsertMany([makeArticle({ id: 'a1', isFavorite: false })]);

      const stored = await ArticleStore.getById('a1');
      expect(stored?.isFavorite).toBe(false);
    });
  });
});

// PR #382 review fix: a failed SQLite open used to poison `articleStore.ts`'s
// module-scoped `dbPromise` forever, and every read propagated that failure
// to its caller. Screens wired their network refresh only inside the read's
// `.then()` with no `.catch`, so a single failed open could strand a screen
// on its initial spinner permanently. These tests exercise a *fresh* module
// instance per case (`jest.resetModules()` + `jest.doMock('expo-sqlite', ...)`
// + `require`), because `dbPromise` is captured at module scope: reusing the
// module imported at the top of this file would already have a healthy,
// resolved `dbPromise` from the tests above, making a fresh open failure
// impossible to simulate.
describe('failure handling (root cause: reads degrade, writes still propagate)', () => {
  type ArticleStoreModule = typeof import('./articleStore');

  // Builds a fresh 'expo-sqlite' mock inline (rather than delegating to
  // __mocks__/expo-sqlite.js) — requiring that file by path from inside a
  // jest.doMock factory for the same module name recurses into Jest's own
  // mock resolution and blows the stack.
  function freshStoreWithFailingOpen(mode: 'always' | 'once'): ArticleStoreModule['ArticleStore'] {
    jest.resetModules();
    jest.doMock('expo-sqlite', () => {
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const { DatabaseSync } = require('node:sqlite');
      let shouldFail = true;
      let nativeDb: InstanceType<typeof DatabaseSync> | null = null;
      return {
        openDatabaseAsync: jest.fn(async () => {
          if (shouldFail) {
            if (mode === 'once') shouldFail = false;
            throw new Error('simulated SQLite open failure');
          }
          if (!nativeDb) nativeDb = new DatabaseSync(':memory:');
          return {
            execAsync: async (sql: string) => nativeDb.exec(sql),
            runAsync: async (sql: string, params: Record<string, unknown> = {}) => {
              const result = nativeDb.prepare(sql).run(params);
              return { changes: result.changes, lastInsertRowId: result.lastInsertRowid };
            },
            getAllAsync: async (sql: string, params: Record<string, unknown> = {}) =>
              nativeDb.prepare(sql).all(params),
            getFirstAsync: async (sql: string, params: Record<string, unknown> = {}) =>
              nativeDb.prepare(sql).get(params) ?? null,
            closeAsync: async () => nativeDb.close(),
          };
        }),
      };
    });
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    return (require('./articleStore') as ArticleStoreModule).ArticleStore;
  }

  let consoleErrorSpy: jest.SpiedFunction<typeof console.error>;

  beforeEach(() => {
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  describe('reads never reject', () => {
    it('listRecent resolves to [] and logs the failure', async () => {
      const store = freshStoreWithFailingOpen('always');
      await expect(store.listRecent(10)).resolves.toEqual([]);
      expect(consoleErrorSpy).toHaveBeenCalled();
    });

    it('listFavorites resolves to [] and logs the failure', async () => {
      const store = freshStoreWithFailingOpen('always');
      await expect(store.listFavorites()).resolves.toEqual([]);
      expect(consoleErrorSpy).toHaveBeenCalled();
    });

    it('getById resolves to null and logs the failure', async () => {
      const store = freshStoreWithFailingOpen('always');
      await expect(store.getById('a1')).resolves.toBeNull();
      expect(consoleErrorSpy).toHaveBeenCalled();
    });
  });

  it('a write method (upsertMany) still rejects on DB failure, unlike reads', async () => {
    const store = freshStoreWithFailingOpen('always');
    await expect(store.upsertMany([makeArticle({ id: 'write-fails' })])).rejects.toThrow(
      'simulated SQLite open failure',
    );
  });

  it('getDb retries after a failed open instead of caching the rejection forever', async () => {
    const store = freshStoreWithFailingOpen('once');

    // First call: the open fails. upsertMany is a write, so it must reject —
    // this also confirms the failure actually happened.
    await expect(store.upsertMany([makeArticle({ id: 'retry-1' })])).rejects.toThrow(
      'simulated SQLite open failure',
    );

    // Second call: without the fix, `dbPromise` would still be the same
    // rejected promise from the first call, and this would reject too.
    await expect(store.upsertMany([makeArticle({ id: 'retry-1' })])).resolves.toBeUndefined();

    const recent = await store.listRecent(10);
    expect(recent.map((a) => a.id)).toEqual(['retry-1']);
  });
});

// task_c55c decision 2: schema changes since v1 (task_a8a4) are applied by a
// PRAGMA user_version-driven migration inside getDb(), not by widening the
// v1 CREATE TABLE — that statement is CREATE TABLE IF NOT EXISTS, so
// widening it is a no-op against every install that already has the v1
// table, and every query naming content_hash would then fail with
// "no such column".
describe('v1 -> v2 migration (content_hash)', () => {
  type ArticleStoreModule = typeof import('./articleStore');

  // The exact v1 shape task_a8a4 shipped — no content_hash column.
  const V1_SCHEMA = `
    CREATE TABLE articles (
      id TEXT PRIMARY KEY,
      url TEXT NOT NULL,
      title TEXT NOT NULL,
      description TEXT,
      image_url TEXT,
      author TEXT,
      published_date TEXT,
      reading_time INTEGER,
      tags TEXT NOT NULL DEFAULT '[]',
      is_read INTEGER NOT NULL DEFAULT 0,
      is_favorite INTEGER NOT NULL DEFAULT 0,
      added_at INTEGER NOT NULL,
      read_at INTEGER,
      scroll_position REAL,
      scroll_fraction REAL,
      body TEXT
    );
  `;

  // Registers a fresh 'expo-sqlite' mock backed by the given node:sqlite
  // instance and requires a fresh articleStore module against it. The
  // DatabaseSync instance lives in the *test's* scope, so — unlike
  // dbPromise — it survives jest.resetModules(): calling this twice with the
  // same nativeDb simulates opening the same on-disk database across two
  // separate app runs, which is exactly what "seed a v1 DB, then run the
  // store" and "a second open is a no-op" need.
  function freshStoreOn(
    nativeDb: import('node:sqlite').DatabaseSync,
  ): ArticleStoreModule['ArticleStore'] {
    jest.resetModules();
    jest.doMock('expo-sqlite', () => ({
      openDatabaseAsync: jest.fn(async () => ({
        execAsync: async (sql: string) => nativeDb.exec(sql),
        runAsync: async (sql: string, params: Record<string, string | number | null> = {}) => {
          const result = nativeDb.prepare(sql).run(params);
          return { changes: result.changes, lastInsertRowId: result.lastInsertRowid };
        },
        getAllAsync: async (sql: string, params: Record<string, string | number | null> = {}) =>
          nativeDb.prepare(sql).all(params),
        getFirstAsync: async (sql: string, params: Record<string, string | number | null> = {}) =>
          nativeDb.prepare(sql).get(params) ?? null,
        closeAsync: async () => nativeDb.close(),
      })),
    }));
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    return (require('./articleStore') as ArticleStoreModule).ArticleStore;
  }

  it('adds content_hash to a DB seeded with the v1 schema and preserves existing rows', async () => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { DatabaseSync } = require('node:sqlite');
    const nativeDb = new DatabaseSync(':memory:') as import('node:sqlite').DatabaseSync;
    nativeDb.exec(V1_SCHEMA);
    nativeDb
      .prepare(
        `INSERT INTO articles (id, url, title, tags, added_at)
         VALUES ($id, $url, $title, $tags, $added_at)`,
      )
      .run({
        $id: 'legacy1',
        $url: 'https://example.com/legacy',
        $title: 'Legacy',
        $tags: '[]',
        $added_at: 100,
      });

    const store = freshStoreOn(nativeDb);
    const legacy = await store.getById('legacy1');

    expect(legacy?.id).toBe('legacy1');
    expect(legacy?.title).toBe('Legacy');
    expect(legacy?.contentHash).toBeUndefined();

    // Query the column directly — proof it exists on the real table, not
    // just tolerated by rowToArticle's `?? undefined`.
    const row = nativeDb
      .prepare('SELECT content_hash FROM articles WHERE id = ?')
      .get('legacy1') as { content_hash: string | null };
    expect(row.content_hash).toBeNull();
  });

  it('does not re-run the ALTER on a second open against an already-migrated DB', async () => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { DatabaseSync } = require('node:sqlite');
    const nativeDb = new DatabaseSync(':memory:') as import('node:sqlite').DatabaseSync;
    nativeDb.exec(V1_SCHEMA);

    const firstOpen = freshStoreOn(nativeDb);
    await firstOpen.upsertMany([makeArticle({ id: 'a1' })]);

    // Simulate an app restart: a fresh module instance (fresh dbPromise),
    // same underlying DB — which the first open already migrated. If
    // migrate() re-ran the ALTER unconditionally instead of checking
    // PRAGMA user_version, this would reject with "duplicate column name:
    // content_hash" instead of resolving.
    const secondOpen = freshStoreOn(nativeDb);
    await expect(secondOpen.getById('a1')).resolves.toMatchObject({ id: 'a1' });
  });
});
