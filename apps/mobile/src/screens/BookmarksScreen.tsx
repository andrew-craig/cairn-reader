import React, { useState, useCallback } from 'react';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { AddLinkModal } from '../components/AddLinkModal';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';

type BookmarksScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Bookmarks'>;

const PAGE_SIZE = 20;

export const BookmarksScreen: React.FC = () => {
  const navigation = useNavigation<BookmarksScreenNavigationProp>();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [offset, setOffset] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);

  const loadBookmarks = useCallback(async (reset = false) => {
    try {
      const currentOffset = reset ? 0 : offset;

      if (reset) {
        setLoading(true);
        setOffset(0);
        setHasMore(true);
      }

      const response = await ReadService.listUserContents({
        is_favorite: true,
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

      setHasMore(response.contents.length === PAGE_SIZE);
      setOffset(currentOffset + response.contents.length);
    } catch (error) {
      console.error('Error loading bookmarks:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
      setLoadingMore(false);
    }
  }, [offset]);

  useFocusEffect(
    useCallback(() => {
      loadBookmarks(true);
    }, [])
  );

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadBookmarks(true);
  }, []);

  const handleLoadMore = useCallback(() => {
    if (!loadingMore && hasMore && !loading) {
      setLoadingMore(true);
      loadBookmarks(false);
    }
  }, [loadingMore, hasMore, loading, loadBookmarks]);

  const handleArticlePress = (article: Article) => {
    navigation.navigate('ArticleDetail', { article });
  };

  const handleAddPress = () => {
    setModalVisible(true);
  };

  const handleAddSuccess = () => {
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
        title="Bookmarks"
        articles={articles}
        loading={loading}
        headerActions={headerActions}
        onArticlePress={handleArticlePress}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        emptyMessage="No bookmarked articles yet"
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
