import AsyncStorage from '@react-native-async-storage/async-storage';
import { StorageService } from './storage';
import { Article } from '../types';

// jest-expo auto-mocks AsyncStorage via the jest setup

const makeArticle = (overrides: Partial<Article> = {}): Article => ({
  id: 'test-1',
  url: 'https://example.com/article',
  title: 'Test Article',
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: Date.now(),
  ...overrides,
});

describe('StorageService', () => {
  beforeEach(async () => {
    await AsyncStorage.clear();
  });

  describe('getArticles', () => {
    it('returns empty array when no articles are stored', async () => {
      const articles = await StorageService.getArticles();
      expect(articles).toEqual([]);
    });

    it('returns stored articles', async () => {
      const article = makeArticle();
      await AsyncStorage.setItem(
        '@cairnreader:articles',
        JSON.stringify([article])
      );

      const articles = await StorageService.getArticles();
      expect(articles).toHaveLength(1);
      expect(articles[0].id).toBe('test-1');
    });
  });

  describe('saveArticles', () => {
    it('persists articles to AsyncStorage', async () => {
      const articles = [makeArticle({ id: 'a' }), makeArticle({ id: 'b' })];
      await StorageService.saveArticles(articles);

      const raw = await AsyncStorage.getItem('@cairnreader:articles');
      expect(raw).not.toBeNull();
      const parsed = JSON.parse(raw!);
      expect(parsed).toHaveLength(2);
      expect(parsed[0].id).toBe('a');
      expect(parsed[1].id).toBe('b');
    });
  });

  describe('addArticle', () => {
    it('prepends article to existing list', async () => {
      const existing = makeArticle({ id: 'existing' });
      await StorageService.saveArticles([existing]);

      const newArticle = makeArticle({ id: 'new' });
      await StorageService.addArticle(newArticle);

      const articles = await StorageService.getArticles();
      expect(articles).toHaveLength(2);
      expect(articles[0].id).toBe('new');
      expect(articles[1].id).toBe('existing');
    });

    it('adds to empty list', async () => {
      const article = makeArticle({ id: 'first' });
      await StorageService.addArticle(article);

      const articles = await StorageService.getArticles();
      expect(articles).toHaveLength(1);
      expect(articles[0].id).toBe('first');
    });
  });

  describe('updateArticle', () => {
    it('updates an existing article by id', async () => {
      const article = makeArticle({ id: 'update-me', title: 'Old Title' });
      await StorageService.saveArticles([article]);

      await StorageService.updateArticle('update-me', { title: 'New Title' });

      const articles = await StorageService.getArticles();
      expect(articles[0].title).toBe('New Title');
      // Other fields should remain
      expect(articles[0].id).toBe('update-me');
    });

    it('does nothing when article id is not found', async () => {
      const article = makeArticle({ id: 'keep' });
      await StorageService.saveArticles([article]);

      await StorageService.updateArticle('nonexistent', { title: 'X' });

      const articles = await StorageService.getArticles();
      expect(articles).toHaveLength(1);
      expect(articles[0].title).toBe('Test Article');
    });

    it('can toggle isFavorite and isRead', async () => {
      const article = makeArticle({ id: 'toggle', isFavorite: false, isRead: false });
      await StorageService.saveArticles([article]);

      await StorageService.updateArticle('toggle', { isFavorite: true, isRead: true });

      const articles = await StorageService.getArticles();
      expect(articles[0].isFavorite).toBe(true);
      expect(articles[0].isRead).toBe(true);
    });
  });

  describe('deleteArticle', () => {
    it('removes an article by id', async () => {
      const articles = [makeArticle({ id: 'a' }), makeArticle({ id: 'b' })];
      await StorageService.saveArticles(articles);

      await StorageService.deleteArticle('a');

      const remaining = await StorageService.getArticles();
      expect(remaining).toHaveLength(1);
      expect(remaining[0].id).toBe('b');
    });

    it('does nothing when id is not found', async () => {
      const articles = [makeArticle({ id: 'a' })];
      await StorageService.saveArticles(articles);

      await StorageService.deleteArticle('nonexistent');

      const remaining = await StorageService.getArticles();
      expect(remaining).toHaveLength(1);
    });
  });

  describe('clearAllArticles', () => {
    it('removes all articles', async () => {
      await StorageService.saveArticles([
        makeArticle({ id: 'a' }),
        makeArticle({ id: 'b' }),
      ]);

      await StorageService.clearAllArticles();

      const articles = await StorageService.getArticles();
      expect(articles).toEqual([]);
    });
  });
});
