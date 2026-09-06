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

  describe('getExploreCache', () => {
    it('returns null when nothing is cached', async () => {
      const cached = await StorageService.getExploreCache();
      expect(cached).toBeNull();
    });

    it('returns the cached explore list', async () => {
      const article = makeArticle();
      await StorageService.saveExploreCache([article]);

      const cached = await StorageService.getExploreCache();
      expect(cached?.articles).toHaveLength(1);
      expect(cached?.articles[0].id).toBe('test-1');
      expect(typeof cached?.cachedAt).toBe('number');
    });
  });

  describe('saveExploreCache', () => {
    it('persists the explore list under the explore cache key', async () => {
      const articles = [makeArticle({ id: 'a' }), makeArticle({ id: 'b' })];
      await StorageService.saveExploreCache(articles);

      const raw = await AsyncStorage.getItem('@cairnreader:explore_cache');
      expect(raw).not.toBeNull();
      const parsed = JSON.parse(raw!);
      expect(parsed.articles).toHaveLength(2);
      expect(parsed.articles[0].id).toBe('a');
      expect(parsed.articles[1].id).toBe('b');
    });
  });
});
