import AsyncStorage from '@react-native-async-storage/async-storage';
import { FeedCache } from './feedCache';
import { Article } from '../types';

const article: Article = {
  id: 'a1',
  url: 'https://example.com/a1',
  title: 'One',
  tags: [],
  isRead: false,
  isFavorite: false,
  list: 'feed',
  addedAt: 1,
};

describe('FeedCache', () => {
  beforeEach(async () => {
    await AsyncStorage.clear();
  });

  it('round-trips a saved page', async () => {
    await FeedCache.save([article]);
    await expect(FeedCache.load()).resolves.toEqual([article]);
  });

  it('loads an empty list when nothing is cached', async () => {
    await expect(FeedCache.load()).resolves.toEqual([]);
  });

  it('loads an empty list when the cached value is corrupt', async () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    await AsyncStorage.setItem('@cairn:feed_cache', '{not json');
    await expect(FeedCache.load()).resolves.toEqual([]);
  });

  it('clear removes the snapshot', async () => {
    await FeedCache.save([article]);
    await FeedCache.clear();
    await expect(FeedCache.load()).resolves.toEqual([]);
  });
});
