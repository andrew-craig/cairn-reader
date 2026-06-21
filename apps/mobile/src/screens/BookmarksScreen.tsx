import React, { useState, useCallback } from 'react';
import { Alert } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { SearchModal } from '../components/SearchModal';
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
  const [cursor, setCursor] = useState<string>('');
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);

  const loadBookmarks = useCallback(async (reset = false) => {
    try {
      const currentCursor = reset ? '' : cursor;

      if (reset) {
        setLoading(true);
        setCursor('');
        setHasMore(true);
      }

      const response = await ReadService.listUserContents({
        is_favorite: true,
        limit: PAGE_SIZE,
        cursor: currentCursor || undefined,
      });

      const transformedArticles = response.contents.map((content) =>
        ReadService.transformToArticle(content)
      );

      if (reset) {
        setArticles(transformedArticles);
      } else {
        setArticles((prev) => [...prev, ...transformedArticles]);
      }

      setHasMore(response.has_more);
      setCursor(response.cursor);
    } catch (error) {
      console.error('Error loading bookmarks:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
      setLoadingMore(false);
    }
  }, [cursor]);

  const searchArticles = async (query: string) => {
    setSearchQuery(query);
    setLoading(true);
    setHasMore(false);

    try {
      const response = await ReadService.searchUserContents({
        q: query,
        limit: PAGE_SIZE,
      });

      const transformedArticles = response.contents.map((content) =>
        ReadService.transformToArticle(content)
      );

      setArticles(transformedArticles);
      setCursor('');
      setHasMore(false);
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
    setCursor('');
    loadBookmarks(true);
  };

  useFocusEffect(
    useCallback(() => {
      if (!searchQuery) {
        loadBookmarks(true);
      }
    }, [searchQuery])
  );

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    if (searchQuery) {
      searchArticles(searchQuery);
    } else {
      loadBookmarks(true);
    }
  }, [searchQuery]);

  const handleLoadMore = useCallback(() => {
    if (!loadingMore && hasMore && !loading && !searchQuery) {
      setLoadingMore(true);
      loadBookmarks(false);
    }
  }, [loadingMore, hasMore, loading, loadBookmarks, searchQuery]);

  const handleArticlePress = (article: Article) => {
    navigation.navigate('ArticleDetail', { article });
  };

  const handleSearchPress = () => {
    setSearchVisible(true);
  };

  const headerActions = (
    <IconButton icon="search-outline" onPress={handleSearchPress} />
  );

  return (
    <>
      <ArticleListScreen
        title="Bookmarks"
        articles={articles}
        loading={loading}
        onBack={() => navigation.goBack()}
        headerActions={headerActions}
        onArticlePress={handleArticlePress}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        emptyMessage={searchQuery ? 'No matching articles' : 'No bookmarked articles yet'}
        onEndReached={handleLoadMore}
        loadingMore={loadingMore}
        searchQuery={searchQuery ?? undefined}
        onClearSearch={clearSearch}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={searchArticles}
      />
    </>
  );
};
