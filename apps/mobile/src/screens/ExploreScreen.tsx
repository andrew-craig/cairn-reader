import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Dimensions, ViewToken } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { SearchModal } from '../components/SearchModal';
import { Article, RootStackParamList } from '../types';
import { ExploreService } from '../services';
import { useAuth } from '../contexts/AuthContext';

type ExploreScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

const MIN_LOOKAHEAD_ARTICLES = 5; // Minimum articles to keep ahead of current scroll position
const ESTIMATED_ARTICLE_HEIGHT = 95; // Estimated height of ArticleRow in pixels
const BUFFER_MULTIPLIER = 1.5; // Load 1.5x screen height worth of articles

// Shown-event batching: queue article IDs locally and POST in batches to
// /api/v1/explore/shown. Flush triggers: queue reaches threshold, debounce
// timer elapses, tab blur, or unmount.
const SHOWN_BATCH_FLUSH_THRESHOLD = 10;
const SHOWN_BATCH_DEBOUNCE_MS = 3000;
const SHOWN_BATCH_MAX_SIZE = 100;

/**
 * Calculate the minimum number of articles needed to fill the screen
 * based on device viewport height and estimated article row height.
 */
const calculateInitialArticleCount = (): number => {
  const screenHeight = Dimensions.get('window').height;
  const articlesPerScreen = Math.ceil(screenHeight / ESTIMATED_ARTICLE_HEIGHT);
  // Multiply by buffer to ensure we have enough to scroll
  const bufferedCount = Math.ceil(articlesPerScreen * BUFFER_MULTIPLIER);
  // Ensure we have at least the minimum lookahead articles
  return Math.max(bufferedCount, MIN_LOOKAHEAD_ARTICLES + 2);
};

export const ExploreScreen: React.FC = () => {
  const navigation = useNavigation<ExploreScreenNavigationProp>();
  const { logout } = useAuth();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);
  const lastVisibleIndexRef = useRef(0);
  const isFetchingRef = useRef(false);
  const hasMoreRef = useRef(true);
  const articlesRef = useRef<Article[]>([]);
  const loadMoreRef = useRef<((minArticles?: number) => Promise<void>) | null>(null);

  // Article-shown tracking. Gate firing on a real scroll interaction so that
  // simply flicking between tabs does not count items as shown.
  const hasUserScrolledRef = useRef(false);
  const shownInSessionRef = useRef<Set<string>>(new Set());
  const shownQueueRef = useRef<string[]>([]);
  const flushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const flushShownRef = useRef<() => void>(() => {});

  const markNoMore = useCallback(() => {
    hasMoreRef.current = false;
    setHasMore(false);
  }, []);

  const loadMoreUntilBuffer = useCallback(async (minArticles?: number) => {
    if (isFetchingRef.current || !hasMoreRef.current) return;

    isFetchingRef.current = true;
    setLoadingMore(true);

    try {
      // Use provided minimum or calculate based on scroll position
      const targetMinimum = minArticles ?? MIN_LOOKAHEAD_ARTICLES;

      // Keep fetching until we have enough articles
      while (true) {
        // Get current articles and lastVisibleIndex from state
        const { count: currentArticlesCount, visibleIndex } = await new Promise<{ count: number; visibleIndex: number }>((resolve) => {
          // Read lastVisibleIndex from ref to avoid stale closure
          const visibleIdx = lastVisibleIndexRef.current;
          setArticles(prev => {
            resolve({ count: prev.length, visibleIndex: visibleIdx });
            return prev;
          });
        });

        // For initial load (minArticles provided), check total count
        // For scroll-based loading, check remaining articles beyond visible position
        const articlesToCheck = minArticles !== undefined
          ? currentArticlesCount
          : currentArticlesCount - visibleIndex;

        // Stop if we have enough articles
        if (articlesToCheck >= targetMinimum) {
          break;
        }

        const newRecommendations = await ExploreService.getRecommendations();

        if (newRecommendations.length === 0) {
          // No more articles available from backend
          markNoMore();
          break;
        }

        // Filter out duplicates and add new articles
        let addedCount = 0;
        await new Promise<void>((resolve) => {
          setArticles(prev => {
            const existingIds = new Set(prev.map(a => a.id));
            const uniqueNew = newRecommendations.filter(a => !existingIds.has(a.id));
            addedCount = uniqueNew.length;
            resolve();
            return [...prev, ...uniqueNew];
          });
        });

        // If we didn't get any new unique articles, break to avoid infinite loop
        if (addedCount === 0) {
          markNoMore();
          break;
        }
      }
    } catch (error) {
      console.error('Error loading more articles:', error);
    } finally {
      setLoadingMore(false);
      isFetchingRef.current = false;
    }
  }, [markNoMore]);

  const loadExploreArticles = useCallback(async () => {
    try {
      const recommendations = await ExploreService.getRecommendations();

      // Calculate how many articles we need
      const minInitialArticles = calculateInitialArticleCount();
      console.log(`Initial article count needed: ${minInitialArticles} (screen height: ${Dimensions.get('window').height}px)`);

      // Set initial articles
      setArticles(recommendations);
      // Also update ref immediately so loadMoreUntilBuffer can see current count
      articlesRef.current = recommendations;

      // If the initial batch was empty, the backend has nothing for us
      if (recommendations.length === 0) {
        markNoMore();
      }

      // If we need more articles, fetch them
      if (recommendations.length < minInitialArticles) {
        // Fetch more batches until we have enough
        let allArticles = [...recommendations];

        while (allArticles.length < minInitialArticles) {
          const moreRecommendations = await ExploreService.getRecommendations();

          if (moreRecommendations.length === 0) {
            markNoMore();
            break;
          }

          // Filter duplicates
          const existingIds = new Set(allArticles.map(a => a.id));
          const uniqueNew = moreRecommendations.filter(a => !existingIds.has(a.id));

          if (uniqueNew.length === 0) {
            markNoMore();
            break;
          }

          allArticles = [...allArticles, ...uniqueNew];
        }

        // Set all articles at once
        setArticles(allArticles);
        articlesRef.current = allArticles;
      }
    } catch (error) {
      console.error('Error loading explore articles:', error);

      // Check if this is a session expiration error
      const errorMessage = error instanceof Error ? error.message : String(error);
      if (errorMessage.includes('Session expired') || errorMessage.includes('Not authenticated')) {
        // Clear articles on auth failure to prevent showing stale content
        setArticles([]);
        articlesRef.current = [];

        // Log out the user to force re-authentication
        console.log('Authentication failed, logging out user');
        await logout();
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [logout, markNoMore]);

  const flushShown = useCallback(() => {
    if (flushTimerRef.current) {
      clearTimeout(flushTimerRef.current);
      flushTimerRef.current = null;
    }
    if (shownQueueRef.current.length === 0) return;
    const batch = shownQueueRef.current.slice(0, SHOWN_BATCH_MAX_SIZE);
    shownQueueRef.current = shownQueueRef.current.slice(batch.length);
    void ExploreService.markShown(batch);
    if (shownQueueRef.current.length > 0) {
      flushTimerRef.current = setTimeout(() => {
        flushShownRef.current();
      }, SHOWN_BATCH_DEBOUNCE_MS);
    }
  }, []);

  useEffect(() => {
    flushShownRef.current = flushShown;
  }, [flushShown]);

  const handleScrollBeginDrag = useRef(() => {
    hasUserScrolledRef.current = true;
  }).current;

  // Keep refs updated with latest values
  useEffect(() => {
    articlesRef.current = articles;
  }, [articles]);

  useEffect(() => {
    loadMoreRef.current = loadMoreUntilBuffer;
  }, [loadMoreUntilBuffer]);

  useEffect(() => {
    loadExploreArticles();
  }, [loadExploreArticles]);

  // Flush pending shown events when the user leaves the Explore tab, and
  // require a fresh scroll interaction on return.
  useEffect(() => {
    const unsubscribe = navigation.addListener('blur', () => {
      flushShownRef.current();
      hasUserScrolledRef.current = false;
    });
    return unsubscribe;
  }, [navigation]);

  useEffect(() => {
    return () => {
      flushShownRef.current();
      if (flushTimerRef.current) {
        clearTimeout(flushTimerRef.current);
        flushTimerRef.current = null;
      }
    };
  }, []);

  const handleRefresh = () => {
    flushShownRef.current();
    setRefreshing(true);
    setArticles([]);
    lastVisibleIndexRef.current = 0;
    hasMoreRef.current = true;
    setHasMore(true);
    hasUserScrolledRef.current = false;
    shownInSessionRef.current = new Set();
    shownQueueRef.current = [];
    loadExploreArticles();
  };

  const handleEndReached = () => {
    if (!hasMoreRef.current) return;
    // Trigger loading more articles when user scrolls near the end
    loadMoreUntilBuffer();
  };

  const handleViewableItemsChanged = useRef((info: {
    viewableItems: ViewToken[];
    changed: ViewToken[];
  }) => {
    if (info.viewableItems.length === 0) return;

    const lastVisible = info.viewableItems[info.viewableItems.length - 1];
    // Use ref to get current articles array
    const currentArticles = articlesRef.current;
    const index = currentArticles.findIndex(a => a.id === lastVisible.item.id);
    if (index !== -1) {
      lastVisibleIndexRef.current = index;

      // Check if we need to load more articles
      const remainingArticles = currentArticles.length - index;
      if (
        remainingArticles < MIN_LOOKAHEAD_ARTICLES &&
        !isFetchingRef.current &&
        hasMoreRef.current
      ) {
        // Use ref to get current loadMore function
        loadMoreRef.current?.();
      }
    }

    // Shown-event detection: only after user has scrolled. Front-half of
    // viewableItems approximates "above the viewport mid-point" since
    // FlatList returns viewable tokens in top-to-bottom order.
    if (!hasUserScrolledRef.current) return;

    const topHalfCount = Math.ceil(info.viewableItems.length / 2);
    for (let i = 0; i < topHalfCount; i++) {
      const articleId = info.viewableItems[i].item.id;
      if (!shownInSessionRef.current.has(articleId)) {
        shownInSessionRef.current.add(articleId);
        shownQueueRef.current.push(articleId);
      }
    }

    if (shownQueueRef.current.length >= SHOWN_BATCH_FLUSH_THRESHOLD) {
      flushShownRef.current();
    } else if (shownQueueRef.current.length > 0 && !flushTimerRef.current) {
      flushTimerRef.current = setTimeout(() => {
        flushShownRef.current();
      }, SHOWN_BATCH_DEBOUNCE_MS);
    }
  }).current;

  const handleArticlePress = async (article: Article) => {
    // Navigate to the Explore article detail screen
    const currentIndex = filteredArticles.findIndex(a => a.id === article.id);
    navigation.navigate('ExploreArticleDetail', { article, articles: filteredArticles, currentIndex });

    // Mark as read in the backend
    try {
      await ExploreService.markAsRead(article.id);

      // Update local state to reflect the read status
      setArticles(prevArticles =>
        prevArticles.map(a =>
          a.id === article.id ? { ...a, isRead: true } : a
        )
      );
    } catch (error) {
      console.error('Error marking article as read:', error);

      // Check if this is a session expiration error
      const errorMessage = error instanceof Error ? error.message : String(error);
      if (errorMessage.includes('Session expired') || errorMessage.includes('Not authenticated')) {
        // Log out the user to force re-authentication
        console.log('Authentication failed while marking article as read, logging out user');
        await logout();
      }
    }
  };

  const handleSearchPress = () => {
    setSearchVisible(true);
  };

  const handleSearch = (query: string) => {
    setSearchQuery(query);
  };

  const clearSearch = () => {
    setSearchQuery(null);
  };

  const filteredArticles = useMemo(() => {
    if (!searchQuery) return articles;
    const q = searchQuery.toLowerCase();
    return articles.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        a.description?.toLowerCase().includes(q) ||
        a.author?.toLowerCase().includes(q)
    );
  }, [articles, searchQuery]);

  const headerActions = (
    <IconButton icon="search-outline" onPress={handleSearchPress} />
  );

  return (
    <>
      <ArticleListScreen
        title="Explore"
        articles={filteredArticles}
        loading={loading}
        headerActions={headerActions}
        onArticlePress={handleArticlePress}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        emptyMessage={searchQuery ? 'No matching articles' : 'No articles available'}
        onEndReached={searchQuery ? undefined : handleEndReached}
        onViewableItemsChanged={searchQuery ? undefined : handleViewableItemsChanged}
        onScrollBeginDrag={searchQuery ? undefined : handleScrollBeginDrag}
        loadingMore={searchQuery ? false : loadingMore}
        endOfListMessage={
          !searchQuery && !hasMore && articles.length > 0
            ? 'Nothing more. Maybe take a walk?'
            : undefined
        }
        searchQuery={searchQuery ?? undefined}
        onClearSearch={clearSearch}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={handleSearch}
      />
    </>
  );
};
