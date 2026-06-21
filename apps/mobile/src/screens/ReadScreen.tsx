import React, { useState, useCallback, useRef } from 'react';
import { Alert } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { AddLinkModal } from '../components/AddLinkModal';
import { SearchModal } from '../components/SearchModal';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';
import { StorageService } from '../services/storage';

// Minimum ms between background refetches triggered by tab focus.
const FOCUS_REFETCH_TTL_MS = 30_000;

type ReadScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

const PAGE_SIZE = 20;

export const ReadScreen: React.FC = () => {
  const navigation = useNavigation<ReadScreenNavigationProp>();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [cursor, setCursor] = useState<string>('');
  const [modalVisible, setModalVisible] = useState(false);
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);
  const [isStale, setIsStale] = useState(false);

  // Timestamp of the last successful network fetch (null = never fetched)
  const lastFetchedAtRef = useRef<number | null>(null);

  const loadReadArticles = useCallback(async (reset = false) => {
    try {
      const currentCursor = reset ? '' : cursor;

      if (reset) {
        setLoading(true);
        setCursor('');
        setHasMore(true);
      }

      const response = await ReadService.listUserContents({
        limit: PAGE_SIZE,
        cursor: currentCursor || undefined,
      });

      const transformedArticles = response.contents.map((content) =>
        ReadService.transformToArticle(content)
      );

      if (reset) {
        setArticles(transformedArticles);
        // Persist fresh data to cache and clear stale indicator
        void StorageService.saveReadListCache(transformedArticles);
        lastFetchedAtRef.current = Date.now();
        setIsStale(false);
      } else {
        setArticles((prev) => [...prev, ...transformedArticles]);
      }

      setHasMore(response.has_more);
      setCursor(response.cursor);
    } catch (error) {
      console.error('Error loading read articles:', error);
      if (reset) {
        // On network failure after showing stale data, mark as stale rather than
        // showing a blocking alert — the user can still read cached content.
        setIsStale(true);
      } else {
        Alert.alert(
          'Error',
          'Failed to load articles. Please try again.',
          [{ text: 'OK' }]
        );
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
      setLoadingMore(false);
    }
  }, [cursor]);

  // On focus: show cached stale data immediately, then refetch in background
  // if TTL has expired.  Skip while a search is active.
  useFocusEffect(
    useCallback(() => {
      if (searchQuery) return;

      const now = Date.now();
      const ttlExpired =
        lastFetchedAtRef.current === null ||
        now - lastFetchedAtRef.current > FOCUS_REFETCH_TTL_MS;

      if (!ttlExpired) return;

      // Show cached data right away so the screen is not blank while fetching
      StorageService.getReadListCache().then((cached) => {
        if (cached && cached.articles.length > 0) {
          setArticles(cached.articles);
          setLoading(false);
          setIsStale(true);
        }
        // Fetch fresh data (will clear isStale on success)
        loadReadArticles(true);
      });
      // We only want this to fire on focus, not on every loadReadArticles
      // identity change, so keep the dependency list minimal.
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [searchQuery])
  );

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
  }, [loadingMore, hasMore, loading, loadReadArticles, searchQuery]);

  const handleArticlePress = (article: Article) => {
    const currentIndex = articles.findIndex(a => a.id === article.id);
    navigation.navigate('ArticleDetail', { article, articles, currentIndex });
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
        staleMessage={isStale ? 'Showing cached data — pull to refresh' : undefined}
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
