import React, { useState, useCallback, useRef } from 'react';
import { Alert, LayoutAnimation } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { ArticleRow } from '../components/common/ArticleRow';
import { SwipeAction } from '../components/common/SwipeAction';
import { ReadIcon } from '../components/icons';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';
import { FeedCache } from '../services/feedCache';
import { ArticleMutations } from '../services/articleMutations';
import { Outbox } from '../services/outbox';
import { invalidateReads } from '../services/readsInvalidation';
import { useCursorArticleList, PAGE_SIZE } from '../hooks/useCursorArticleList';

// Minimum ms between background refetches triggered by tab focus.
const FOCUS_REFETCH_TTL_MS = 30_000;

type FeedScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

/**
 * The Feed list: items from sources the user routed to Feed, to skim. Online
 * only, except Save to Reads: that goes through ArticleMutations.saveToReads,
 * which stores the item as a Reads item and queues the move in the outbox
 * when offline. The last page is kept in FeedCache purely so a failed refresh
 * can still show something (with a stale banner).
 */
export const FeedScreen: React.FC = () => {
  const navigation = useNavigation<FeedScreenNavigationProp>();
  const [isStale, setIsStale] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Timestamp of the last successful network fetch (null = never fetched)
  const lastFetchedAtRef = useRef<number | null>(null);

  const fetchPage = useCallback(
    (cursor: string | undefined) =>
      ReadService.listUserContents({ limit: PAGE_SIZE, cursor, list: 'feed' }),
    [],
  );

  const onResetLoaded = useCallback((next: Article[]) => {
    FeedCache.save(next).catch((err) => console.error('Failed to cache feed:', err));
    lastFetchedAtRef.current = Date.now();
    setIsStale(false);
    setError(null);
  }, []);

  const onLoadError = useCallback((reset: boolean) => {
    if (reset) {
      // Mark stale rather than blocking when there is something to show;
      // ArticleListScreen surfaces `error` + Retry only when the list is empty.
      setIsStale(true);
      setError("Couldn't load your Feed. Check your connection and try again.");
    } else {
      Alert.alert('Error', 'Failed to load articles. Please try again.', [{ text: 'OK' }]);
    }
  }, []);

  const {
    articles,
    setArticles,
    loading,
    setLoading,
    refreshing,
    loadingMore,
    searchOpen,
    query,
    setQuery,
    searchQuery,
    openSearch,
    closeSearch,
    load,
    handleRefresh,
    handleLoadMore,
  } = useCursorArticleList({
    fetchPage,
    onResetLoaded,
    onLoadError,
    list: 'feed',
    // Until a queued Save to Reads drains, the server still lists the item in
    // Feed; leave it out so a refetch or search doesn't bring it back.
    exclude: Outbox.pendingSaveIds,
  });

  const articlesRef = useRef<Article[]>(articles);
  articlesRef.current = articles;

  // On focus: refetch once the TTL has lapsed. If nothing is on screen yet,
  // prime from the cached last page first so a failed fetch isn't a blank tab.
  useFocusEffect(
    useCallback(() => {
      if (searchQuery) return;

      const now = Date.now();
      const ttlExpired =
        lastFetchedAtRef.current === null || now - lastFetchedAtRef.current > FOCUS_REFETCH_TTL_MS;
      if (!ttlExpired) return;

      const refetch = () => void load(true);
      if (articlesRef.current.length > 0) {
        refetch();
        return;
      }
      FeedCache.load().then((cached) => {
        if (cached.length > 0 && articlesRef.current.length === 0) {
          setArticles(cached);
          setLoading(false);
          setIsStale(true);
        }
        refetch();
      });
    }, [searchQuery, load, setArticles, setLoading]),
  );

  // Called by the reader when an item leaves the Feed (Save to Reads).
  const handleLeftFeed = useCallback(
    (articleId: string) => {
      const next = articlesRef.current.filter((a) => a.id !== articleId);
      setArticles(next);
      // While searching, `next` is a subset of search results, not the Feed
      // page; the quiet refresh when search closes re-caches the page.
      if (!searchQuery) {
        FeedCache.save(next).catch((err) => console.error('Failed to cache feed:', err));
      }
    },
    [setArticles, searchQuery],
  );

  // Awaited, like the reader's Save to Reads: the row leaves the Feed once the
  // server has moved it or the move is queued offline. Only a definitive
  // rejection rethrows, which makes SwipeAction spring the row back.
  const handleSwipeSaveToReads = async (article: Article) => {
    try {
      await ArticleMutations.saveToReads(article);
    } catch (error) {
      console.error('Failed to save article to Reads:', error);
      Alert.alert('Error', 'Failed to save to Reads');
      throw error;
    }
    invalidateReads();
    // Collapse the gap the swiped-away row leaves behind.
    LayoutAnimation.configureNext(LayoutAnimation.Presets.easeInEaseOut);
    handleLeftFeed(article.id);
  };

  const handleArticlePress = (article: Article) => {
    const currentIndex = articles.findIndex((a) => a.id === article.id);
    navigation.navigate('ArticleDetail', {
      article,
      articles,
      currentIndex,
      onArchived: handleLeftFeed,
    });
  };

  return (
    <ArticleListScreen
      title="Feed"
      articles={articles}
      loading={loading}
      renderItem={(article) => (
        <SwipeAction direction="right" icon={ReadIcon} color="info" onAction={() => handleSwipeSaveToReads(article)}>
          <ArticleRow article={article} onPress={() => handleArticlePress(article)} />
        </SwipeAction>
      )}
      onRefresh={handleRefresh}
      refreshing={refreshing}
      emptyMessage={searchQuery ? 'No matching articles' : 'Your Feed is empty'}
      onEndReached={handleLoadMore}
      loadingMore={loadingMore}
      search={{
        open: searchOpen,
        query,
        onChangeQuery: setQuery,
        onOpen: openSearch,
        onClose: closeSearch,
        placeholder: 'Search your feed',
      }}
      staleMessage={isStale ? 'Showing cached data — pull to refresh' : undefined}
      error={error}
      onRetry={() => load(true)}
    />
  );
};
