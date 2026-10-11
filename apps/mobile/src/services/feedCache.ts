import AsyncStorage from '@react-native-async-storage/async-storage';
import { Article } from '../types';

const FEED_CACHE_KEY = '@cairn:feed_cache';

/**
 * Snapshot of the last Feed page, shown (with a stale banner) when a refresh
 * fails. Feed is online-only: this is deliberately not ArticleStore, so Feed
 * items never reach the offline Reads store, its prefetch or the outbox.
 * Cleared on logout like ArticleStore.
 */
export const FeedCache = {
  async save(articles: Article[]): Promise<void> {
    await AsyncStorage.setItem(FEED_CACHE_KEY, JSON.stringify(articles));
  },

  async load(): Promise<Article[]> {
    try {
      const raw = await AsyncStorage.getItem(FEED_CACHE_KEY);
      return raw ? (JSON.parse(raw) as Article[]) : [];
    } catch (error) {
      console.error('Error loading cached feed:', error);
      return [];
    }
  },

  async clear(): Promise<void> {
    await AsyncStorage.removeItem(FEED_CACHE_KEY);
  },
};
