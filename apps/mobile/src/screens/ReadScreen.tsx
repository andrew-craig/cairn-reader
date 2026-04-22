import React, { useState, useCallback } from 'react';
import { Alert } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { AddLinkModal } from '../components/AddLinkModal';
import { SearchModal } from '../components/SearchModal';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';

type ReadScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

const PAGE_SIZE = 20;

export const ReadScreen: React.FC = () => {
  const navigation = useNavigation<ReadScreenNavigationProp>();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [offset, setOffset] = useState(0);
  const [modalVisible, setModalVisible] = useState(false);
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);

  useFocusEffect(
    useCallback(() => {
      if (!searchQuery) {
        loadReadArticles(true);
      }
    }, [searchQuery])
  );

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

  const searchArticles = async (query: string) => {
    setSearchQuery(query);
    setLoading(true);
    setHasMore(false);

    try {
      const response = await ReadService.searchUserContents({
        q: query,
        limit: PAGE_SIZE,
        offset: 0,
      });

      const transformedArticles = response.contents.map((content) =>
        ReadService.transformToArticle(content)
      );

      setArticles(transformedArticles);
      setOffset(transformedArticles.length);
      setHasMore(response.contents.length === PAGE_SIZE);
    } catch (error) {
      console.error('Error searching articles:', error);
      Alert.alert('Error', 'Failed to search articles. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const clearSearch = () => {
    setSearchQuery(null);
    setArticles([]);
    setOffset(0);
    loadReadArticles(true);
  };

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    if (searchQuery) {
      searchArticles(searchQuery);
    } else {
      loadReadArticles(true);
    }
  }, [searchQuery]);

  const handleLoadMore = useCallback(() => {
    if (!loadingMore && hasMore && !loading && !searchQuery) {
      setLoadingMore(true);
      loadReadArticles(false);
    }
  }, [loadingMore, hasMore, loading, offset, searchQuery]);

  const handleArticlePress = (article: Article) => {
    navigation.navigate('ArticleDetail', { article });
  };

  const handleAddPress = () => {
    setModalVisible(true);
  };

  const handleAddSuccess = () => {
    // Refresh the list to show the new article or subscription
    handleRefresh();
  };

  const handleSearchPress = () => {
    setSearchVisible(true);
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
        emptyMessage={searchQuery ? 'No matching articles' : 'No saved articles yet'}
        onEndReached={handleLoadMore}
        loadingMore={loadingMore}
        searchQuery={searchQuery ?? undefined}
        onClearSearch={clearSearch}
      />
      <AddLinkModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        onSuccess={handleAddSuccess}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={searchArticles}
      />
    </>
  );
};
