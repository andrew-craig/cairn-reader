import React, { useState, useEffect, useCallback } from 'react';
import { Alert } from 'react-native';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { AddLinkModal } from '../components/AddLinkModal';
import { Article } from '../types';
import { ReadService } from '../services/read';

const PAGE_SIZE = 20;

export const ReadScreen: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [offset, setOffset] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);

  useEffect(() => {
    loadReadArticles();
  }, []);

  const loadReadArticles = async (reset = false) => {
    try {
      const currentOffset = reset ? 0 : offset;

      if (reset) {
        setLoading(true);
        setOffset(0);
        setHasMore(true);
      }

      const response = await ReadService.listUserContents({
        limit: PAGE_SIZE,
        offset: currentOffset,
      });

      const transformedArticles = response.contents.map((content) =>
        ReadService.transformToArticle(content)
      );

      if (reset) {
        setArticles(transformedArticles);
      } else {
        setArticles((prev) => [...prev, ...transformedArticles]);
      }

      // Check if there are more items to load
      setHasMore(response.contents.length === PAGE_SIZE);
      setOffset(currentOffset + response.contents.length);
    } catch (error) {
      console.error('Error loading read articles:', error);
      Alert.alert(
        'Error',
        'Failed to load articles. Please try again.',
        [{ text: 'OK' }]
      );
    } finally {
      setLoading(false);
      setRefreshing(false);
      setLoadingMore(false);
    }
  };

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadReadArticles(true);
  }, []);

  const handleLoadMore = useCallback(() => {
    if (!loadingMore && hasMore && !loading) {
      setLoadingMore(true);
      loadReadArticles(false);
    }
  }, [loadingMore, hasMore, loading, offset]);

  const handleArticlePress = (article: Article) => {
    // TODO: Navigate to article detail screen
    console.log('Article pressed:', article.id);
  };

  const handleAddPress = () => {
    setModalVisible(true);
  };

  const handleAddSuccess = () => {
    // Refresh the list to show the new article or subscription
    handleRefresh();
  };

  const handleSearchPress = () => {
    // TODO: Navigate to search screen
    console.log('Search pressed');
  };

  const headerActions = (
    <>
      <IconButton icon="add-outline" onPress={handleAddPress} />
      <IconButton icon="search-outline" onPress={handleSearchPress} />
    </>
  );

  return (
    <>
      <ArticleListScreen
        title="Read"
        articles={articles}
        loading={loading}
        headerActions={headerActions}
        onArticlePress={handleArticlePress}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        emptyMessage="No saved articles yet"
        onEndReached={handleLoadMore}
        loadingMore={loadingMore}
      />
      <AddLinkModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        onSuccess={handleAddSuccess}
      />
    </>
  );
};
