import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Dimensions, ViewToken } from 'react-native';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { Article } from '../types';
import { ExploreService } from '../services';

const MIN_LOOKAHEAD_ARTICLES = 5; // Minimum articles to keep ahead of current scroll position
const ESTIMATED_ARTICLE_HEIGHT = 95; // Estimated height of ArticleRow in pixels
const BUFFER_MULTIPLIER = 1.5; // Load 1.5x screen height worth of articles

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
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [lastVisibleIndex, setLastVisibleIndex] = useState(0);
  const isFetchingRef = useRef(false);
  const articlesRef = useRef<Article[]>([]);
  const loadMoreRef = useRef<((minArticles?: number) => Promise<void>) | null>(null);

  const loadMoreUntilBuffer = useCallback(async (minArticles?: number) => {
    if (isFetchingRef.current) return;

    isFetchingRef.current = true;
    setLoadingMore(true);

    try {
      let previousLength = 0;

      // Use provided minimum or calculate based on scroll position
      const targetMinimum = minArticles ?? MIN_LOOKAHEAD_ARTICLES;

      // Keep fetching until we have enough articles
      while (true) {
        // Get current articles count to check buffer
        const currentArticlesCount = await new Promise<number>((resolve) => {
          setArticles(prev => {
            resolve(prev.length);
            return prev;
          });
        });

        // For initial load (minArticles provided), check total count
        // For scroll-based loading, check remaining articles beyond visible position
        const articlesToCheck = minArticles !== undefined
          ? currentArticlesCount
          : currentArticlesCount - lastVisibleIndex;

        // Stop if we have enough articles
        if (articlesToCheck >= targetMinimum) {
          break;
        }

        const newRecommendations = await ExploreService.getRecommendations();

        if (newRecommendations.length === 0) {
          // No more articles available from backend
          break;
        }

        // Filter out duplicates and add new articles
        let addedCount = 0;
        setArticles(prev => {
          const existingIds = new Set(prev.map(a => a.id));
          const uniqueNew = newRecommendations.filter(a => !existingIds.has(a.id));
          addedCount = uniqueNew.length;
          return [...prev, ...uniqueNew];
        });

        // If we didn't get any new unique articles, break to avoid infinite loop
        if (addedCount === 0) {
          break;
        }

        // Safety check: if articles count hasn't changed, break
        if (currentArticlesCount === previousLength) {
          break;
        }
        previousLength = currentArticlesCount;
      }
    } catch (error) {
      console.error('Error loading more articles:', error);
    } finally {
      setLoadingMore(false);
      isFetchingRef.current = false;
    }
  }, [lastVisibleIndex]);

  const loadExploreArticles = useCallback(async () => {
    try {
      const recommendations = await ExploreService.getRecommendations();
      setArticles(recommendations);

      // After initial load, ensure we have enough articles to fill the screen
      // Calculate dynamically based on viewport height
      const minInitialArticles = calculateInitialArticleCount();
      console.log(`Initial article count needed: ${minInitialArticles} (screen height: ${Dimensions.get('window').height}px)`);

      if (recommendations.length < minInitialArticles) {
        await loadMoreUntilBuffer(minInitialArticles);
      }
    } catch (error) {
      console.error('Error loading explore articles:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [loadMoreUntilBuffer]);

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

  const handleRefresh = () => {
    setRefreshing(true);
    setArticles([]);
    setLastVisibleIndex(0);
    loadExploreArticles();
  };

  const handleEndReached = () => {
    // Trigger loading more articles when user scrolls near the end
    loadMoreUntilBuffer();
  };

  const handleViewableItemsChanged = useRef((info: {
    viewableItems: ViewToken[];
    changed: ViewToken[];
  }) => {
    if (info.viewableItems.length > 0) {
      const lastVisible = info.viewableItems[info.viewableItems.length - 1];
      // Use ref to get current articles array
      const currentArticles = articlesRef.current;
      const index = currentArticles.findIndex(a => a.id === lastVisible.item.id);
      if (index !== -1) {
        setLastVisibleIndex(index);

        // Check if we need to load more articles
        const remainingArticles = currentArticles.length - index;
        if (remainingArticles < MIN_LOOKAHEAD_ARTICLES && !isFetchingRef.current) {
          // Use ref to get current loadMore function
          loadMoreRef.current?.();
        }
      }
    }
  }).current;

  const handleArticlePress = async (article: Article) => {
    // TODO: Navigate to article detail screen or open in browser
    console.log('Article pressed:', article.id);

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
    }
  };

  const handleSearchPress = () => {
    // TODO: Navigate to search screen
    console.log('Search pressed');
  };

  const headerActions = (
    <IconButton icon="search-outline" onPress={handleSearchPress} />
  );

  return (
    <ArticleListScreen
      title="Explore"
      articles={articles}
      loading={loading}
      headerActions={headerActions}
      onArticlePress={handleArticlePress}
      onRefresh={handleRefresh}
      refreshing={refreshing}
      emptyMessage="No articles available"
      onEndReached={handleEndReached}
      onViewableItemsChanged={handleViewableItemsChanged}
      loadingMore={loadingMore}
    />
  );
};
