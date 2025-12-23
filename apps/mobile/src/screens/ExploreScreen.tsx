import React, { useState, useEffect, useRef } from 'react';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { Article } from '../types';
import { ExploreService } from '../services';

const MIN_LOOKAHEAD_ARTICLES = 5; // Minimum articles to keep ahead of current scroll position

export const ExploreScreen: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [lastVisibleIndex, setLastVisibleIndex] = useState(0);
  const isFetchingRef = useRef(false);

  useEffect(() => {
    loadExploreArticles();
  }, []);

  const loadExploreArticles = async () => {
    try {
      const recommendations = await ExploreService.getRecommendations();
      setArticles(recommendations);

      // After initial load, check if we need more articles
      if (recommendations.length < MIN_LOOKAHEAD_ARTICLES) {
        await loadMoreUntilBuffer();
      }
    } catch (error) {
      console.error('Error loading explore articles:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = () => {
    setRefreshing(true);
    setArticles([]);
    setLastVisibleIndex(0);
    loadExploreArticles();
  };

  const loadMoreUntilBuffer = async () => {
    if (isFetchingRef.current) return;

    isFetchingRef.current = true;
    setLoadingMore(true);

    try {
      let shouldContinue = true;
      let previousLength = 0;

      // Keep fetching until we have at least MIN_LOOKAHEAD_ARTICLES beyond the visible position
      while (shouldContinue) {
        // Get current articles count to check buffer
        const currentArticlesCount = await new Promise<number>((resolve) => {
          setArticles(prev => {
            resolve(prev.length);
            return prev;
          });
        });

        const remainingArticles = currentArticlesCount - lastVisibleIndex;

        // Stop if we have enough articles in the buffer
        if (remainingArticles >= MIN_LOOKAHEAD_ARTICLES) {
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
  };

  const handleEndReached = () => {
    // Trigger loading more articles when user scrolls near the end
    loadMoreUntilBuffer();
  };

  const handleViewableItemsChanged = useRef(({ viewableItems }: any) => {
    if (viewableItems.length > 0) {
      const lastVisible = viewableItems[viewableItems.length - 1];
      const index = articles.findIndex(a => a.id === lastVisible.item.id);
      if (index !== -1) {
        setLastVisibleIndex(index);

        // Check if we need to load more articles
        const remainingArticles = articles.length - index;
        if (remainingArticles < MIN_LOOKAHEAD_ARTICLES && !isFetchingRef.current) {
          loadMoreUntilBuffer();
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
