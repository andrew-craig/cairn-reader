import AsyncStorage from '@react-native-async-storage/async-storage';
import { Article } from '../types';

const EXPLORE_CACHE_KEY = '@cairnreader:explore_cache';

interface CachedList {
  articles: Article[];
  cachedAt: number; // Unix ms
}

export const StorageService = {
  // --- Stale-while-revalidate cache for the Explore feed ---

  async getExploreCache(): Promise<CachedList | null> {
    try {
      const json = await AsyncStorage.getItem(EXPLORE_CACHE_KEY);
      return json ? (JSON.parse(json) as CachedList) : null;
    } catch {
      return null;
    }
  },

  async saveExploreCache(articles: Article[]): Promise<void> {
    try {
      const entry: CachedList = { articles, cachedAt: Date.now() };
      await AsyncStorage.setItem(EXPLORE_CACHE_KEY, JSON.stringify(entry));
    } catch (error) {
      console.error('Error saving explore cache:', error);
    }
  },
};
