import { ArticleStore } from './articleStore';
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
  });
});
