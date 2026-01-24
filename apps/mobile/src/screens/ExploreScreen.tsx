import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Dimensions, ViewToken } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { Article, RootStackParamList } from '../types';
import { ExploreService } from '../services';
import { useAuth } from '../contexts/AuthContext';

type ExploreScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

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
  const navigation = useNavigation<ExploreScreenNavigationProp>();
  const { logout } = useAuth();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const lastVisibleIndexRef = useRef(0);
  const isFetchingRef = useRef(false);
  const articlesRef = useRef<Article[]>([]);
  const loadMoreRef = useRef<((minArticles?: number) => Promise<void>) | null>(null);

  const loadMoreUntilBuffer = useCallback(async (minArticles?: number) => {
    if (isFetchingRef.current) return;

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
          break;
        }
      }
    } catch (error) {
      console.error('Error loading more articles:', error);
    } finally {
      setLoadingMore(false);
      isFetchingRef.current = false;
    }
  }, []); // No dependencies - uses refs for mutable values

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

      // If we need more articles, fetch them
      if (recommendations.length < minInitialArticles) {
        // Fetch more batches until we have enough
        let allArticles = [...recommendations];

        while (allArticles.length < minInitialArticles) {
          const moreRecommendations = await ExploreService.getRecommendations();

          if (moreRecommendations.length === 0) {
            break;
          }

          // Filter duplicates
          const existingIds = new Set(allArticles.map(a => a.id));
          const uniqueNew = moreRecommendations.filter(a => !existingIds.has(a.id));

          if (uniqueNew.length === 0) {
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
  }, [logout]);

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
    lastVisibleIndexRef.current = 0;
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
        lastVisibleIndexRef.current = index;

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
    // Navigate to the Explore article detail screen
    navigation.navigate('ExploreArticleDetail', { article });

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
