import { useCallback, useRef, useState } from 'react';
import { Alert } from 'react-native';
import { Article } from '../types';
import { ReadService } from '../services/read';
import { UserContentsListResponse } from '@cairn/shared';

/** Reading-list / bookmarks page size. */
export const PAGE_SIZE = 20;

interface CursorArticleListSource {
  /** Fetch one page. `cursor` is undefined for the first page. */
  fetchPage: (cursor: string | undefined) => Promise<UserContentsListResponse>;
  /** Called with the full first page after a reset load succeeds (e.g. to cache it). */
  onResetLoaded?: (articles: Article[]) => void;
  /**
   * Called when a load fails. `reset` distinguishes a first-page/refresh failure
   * from a load-more failure. When omitted, the error is only logged.
   */
  onLoadError?: (reset: boolean) => void;
}

/**
 * The cursor-paginated article-list state machine shared by ReadScreen and
 * BookmarksScreen: items + cursor + hasMore + loading/loadingMore/refreshing,
 * reset-vs-append, and the search overlay (both screens search via
 * ReadService.searchUserContents). Screen-specific concerns — cache priming,
 * stale banners, focus-refetch TTL, archive mutation — stay in the screens and
 * use the exposed `setArticles` / `setLoading` / `load`.
 *
 * The cursor is held in a ref (mirroring apps/web/src/routes/Read.tsx) so `load`
 * keeps a stable identity and can be called safely from a focus effect without
 * re-firing on every page.
 */
export function useCursorArticleList({
  fetchPage,
  onResetLoaded,
  onLoadError,
}: CursorArticleListSource) {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);

  const cursorRef = useRef<string>('');

  const load = useCallback(
    async (reset = false) => {
      try {
        if (reset) {
          setLoading(true);
          cursorRef.current = '';
          setHasMore(true);
        }

        const response = await fetchPage(cursorRef.current || undefined);
        const page = response.contents.map((c) => ReadService.transformToArticle(c));

        if (reset) {
          setArticles(page);
          onResetLoaded?.(page);
        } else {
          setArticles((prev) => [...prev, ...page]);
        }

        setHasMore(response.has_more);
        cursorRef.current = response.cursor;
      } catch (error) {
        console.error('Error loading articles:', error);
        onLoadError?.(reset);
      } finally {
        setLoading(false);
        setRefreshing(false);
        setLoadingMore(false);
      }
    },
    [fetchPage, onResetLoaded, onLoadError],
  );

  const search = useCallback(async (query: string) => {
    setSearchQuery(query);
    setLoading(true);
    setHasMore(false);

    try {
      const response = await ReadService.searchUserContents({ q: query, limit: PAGE_SIZE });
      setArticles(response.contents.map((c) => ReadService.transformToArticle(c)));
      cursorRef.current = '';
      setHasMore(false);
    } catch (error) {
      console.error('Error searching articles:', error);
      Alert.alert('Error', 'Failed to search articles. Please try again.');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  const clearSearch = useCallback(() => {
    setSearchQuery(null);
    setArticles([]);
    cursorRef.current = '';
    void load(true);
  }, [load]);

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    if (searchQuery) {
      void search(searchQuery);
    } else {
      void load(true);
    }
  }, [searchQuery, search, load]);

  const handleLoadMore = useCallback(() => {
    if (!loadingMore && hasMore && !loading && !searchQuery) {
      setLoadingMore(true);
      void load(false);
    }
  }, [loadingMore, hasMore, loading, searchQuery, load]);

  return {
    articles,
    setArticles,
    loading,
    setLoading,
    refreshing,
    loadingMore,
    hasMore,
    searchQuery,
    load,
    search,
    clearSearch,
    handleRefresh,
    handleLoadMore,
  };
}
