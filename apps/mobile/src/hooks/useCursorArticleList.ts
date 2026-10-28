import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert } from 'react-native';
import { Article } from '../types';
import { ReadService } from '../services/read';
import { ContentList, UserContentsListResponse } from '@cairn/shared';

/** Reading-list / bookmarks page size. */
export const PAGE_SIZE = 20;

/** Live search fires this long after the user stops typing (matches web). */
const SEARCH_DEBOUNCE_MS = 300;

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
  /** Scopes search to one list. `fetchPage` stays responsible for scoping the page fetch. */
  list?: ContentList;
}

/**
 * The cursor-paginated article-list state machine shared by the Reads, Feed and
 * Bookmarks screens: items + cursor + hasMore + loading/loadingMore/refreshing,
 * reset-vs-append, and the header search: open state, query, and a debounced
 * live search via ReadService.searchUserContents. Results replace the list
 * while a query is held; clearing it restores the pre-search list at once and
 * refreshes it quietly, so the screen never swaps to a full-screen spinner
 * under the search field. Screen-specific concerns — cache priming,
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
  list,
}: CursorArticleListSource) {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [searchOpen, setSearchOpen] = useState(false);
  const [query, setQuery] = useState('');
  const searchQuery = query.trim() || null;

  const cursorRef = useRef<string>('');
  const articlesRef = useRef<Article[]>(articles);
  articlesRef.current = articles;
  const hasMoreRef = useRef(hasMore);
  hasMoreRef.current = hasMore;
  // The list as it was when the current search started; null when no search
  // results are showing.
  const preSearchRef = useRef<{ articles: Article[]; cursor: string; hasMore: boolean } | null>(null);

  const load = useCallback(
    async (reset = false, quiet = false) => {
      try {
        if (reset) {
          if (!quiet) setLoading(true);
          cursorRef.current = '';
          setHasMore(true);
        }

        const response = await fetchPage(cursorRef.current || undefined);
        const page = response.contents.map((c) => ReadService.transformToArticle(c));

        // A reset that lands while search results are showing refreshes the
        // list they will restore to, rather than replacing the results.
        if (reset && preSearchRef.current) {
          preSearchRef.current = { articles: page, cursor: response.cursor, hasMore: response.has_more };
          onResetLoaded?.(page);
          return;
        }

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

  const runSearch = useCallback(
    async (q: string, isCancelled: () => boolean = () => false) => {
      try {
        const response = await ReadService.searchUserContents({ q, limit: PAGE_SIZE, list });
        if (isCancelled()) return;
        setArticles(response.contents.map((c) => ReadService.transformToArticle(c)));
        cursorRef.current = '';
        setHasMore(false);
      } catch (error) {
        if (isCancelled()) return;
        console.error('Error searching articles:', error);
        Alert.alert('Error', 'Failed to search articles. Please try again.');
      } finally {
        if (!isCancelled()) setRefreshing(false);
      }
    },
    [list],
  );

  useEffect(() => {
    if (!searchQuery) {
      const saved = preSearchRef.current;
      if (saved) {
        preSearchRef.current = null;
        setArticles(saved.articles);
        cursorRef.current = saved.cursor;
        setHasMore(saved.hasMore);
        void load(true, true);
      }
      return;
    }

    let cancelled = false;
    const timerId = setTimeout(() => {
      if (!preSearchRef.current) {
        preSearchRef.current = {
          articles: articlesRef.current,
          cursor: cursorRef.current,
          hasMore: hasMoreRef.current,
        };
      }
      void runSearch(searchQuery, () => cancelled);
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      clearTimeout(timerId);
    };
  }, [searchQuery, runSearch, load]);

  const openSearch = useCallback(() => setSearchOpen(true), []);

  const closeSearch = useCallback(() => {
    setSearchOpen(false);
    setQuery('');
  }, []);

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    if (searchQuery) {
      void runSearch(searchQuery);
    } else {
      void load(true);
    }
  }, [searchQuery, runSearch, load]);

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
    searchOpen,
    query,
    setQuery,
    searchQuery,
    openSearch,
    closeSearch,
    load,
    handleRefresh,
    handleLoadMore,
  };
}
