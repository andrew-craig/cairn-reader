import AsyncStorage from '@react-native-async-storage/async-storage';
import { Article } from '../types';

const ARTICLES_KEY = '@cairnreader:articles';
const EXPLORE_CACHE_KEY = '@cairnreader:explore_cache';
const READ_LIST_CACHE_KEY = '@cairnreader:read_list_cache';

interface CachedList {
  articles: Article[];
  cachedAt: number; // Unix ms
}

export const StorageService = {
  async getArticles(): Promise<Article[]> {
    try {
      const articlesJson = await AsyncStorage.getItem(ARTICLES_KEY);
      return articlesJson ? JSON.parse(articlesJson) : [];
    } catch (error) {
      console.error('Error loading articles:', error);
      return [];
    }
  },

  async saveArticles(articles: Article[]): Promise<void> {
    try {
      await AsyncStorage.setItem(ARTICLES_KEY, JSON.stringify(articles));
    } catch (error) {
      console.error('Error saving articles:', error);
      throw error;
    }
  },

  async addArticle(article: Article): Promise<void> {
    try {
      const articles = await this.getArticles();
      articles.unshift(article);
      await this.saveArticles(articles);
    } catch (error) {
      console.error('Error adding article:', error);
      throw error;
    }
  },

  async updateArticle(id: string, updates: Partial<Article>): Promise<void> {
    try {
      const articles = await this.getArticles();
      const index = articles.findIndex(a => a.id === id);
      if (index !== -1) {
        articles[index] = { ...articles[index], ...updates };
        await this.saveArticles(articles);
      }
    } catch (error) {
      console.error('Error updating article:', error);
      throw error;
    }
  },

  async deleteArticle(id: string): Promise<void> {
    try {
      const articles = await this.getArticles();
      const filtered = articles.filter(a => a.id !== id);
      await this.saveArticles(filtered);
    } catch (error) {
      console.error('Error deleting article:', error);
      throw error;
    }
  },

  async clearAllArticles(): Promise<void> {
    try {
      await AsyncStorage.removeItem(ARTICLES_KEY);
    } catch (error) {
      console.error('Error clearing articles:', error);
      throw error;
    }
  },

  // --- Stale-while-revalidate caches for list screens ---

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

  async getReadListCache(): Promise<CachedList | null> {
    try {
      const json = await AsyncStorage.getItem(READ_LIST_CACHE_KEY);
      return json ? (JSON.parse(json) as CachedList) : null;
    } catch {
      return null;
    }
  },

  async saveReadListCache(articles: Article[]): Promise<void> {
    try {
      const entry: CachedList = { articles, cachedAt: Date.now() };
      await AsyncStorage.setItem(READ_LIST_CACHE_KEY, JSON.stringify(entry));
    } catch (error) {
      console.error('Error saving read list cache:', error);
    }
  },
};
