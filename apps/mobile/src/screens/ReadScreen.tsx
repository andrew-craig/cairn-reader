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
import { useCursorArticleList, PAGE_SIZE } from '../hooks/useCursorArticleList';

// Minimum ms between background refetches triggered by tab focus.
const FOCUS_REFETCH_TTL_MS = 30_000;

type ReadScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

export const ReadScreen: React.FC = () => {
  const navigation = useNavigation<ReadScreenNavigationProp>();
  const [modalVisible, setModalVisible] = useState(false);
  const [searchVisible, setSearchVisible] = useState(false);
  const [isStale, setIsStale] = useState(false);

  // Timestamp of the last successful network fetch (null = never fetched)
  const lastFetchedAtRef = useRef<number | null>(null);

  const fetchPage = useCallback(
    (cursor: string | undefined) => ReadService.listUserContents({ limit: PAGE_SIZE, cursor }),
    [],
  );

  const onResetLoaded = useCallback((next: Article[]) => {
    void StorageService.saveReadListCache(next);
    lastFetchedAtRef.current = Date.now();
    setIsStale(false);
  }, []);

  const onLoadError = useCallback((reset: boolean) => {
    if (reset) {
      // Network failure after showing stale data — mark stale rather than
      // blocking with an alert; the user can still read cached content.
      setIsStale(true);
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
    searchQuery,
    load,
    search,
    clearSearch,
    handleRefresh,
    handleLoadMore,
  } = useCursorArticleList({ fetchPage, onResetLoaded, onLoadError });

  // On focus: show cached stale data immediately, then refetch in the
  // background if the TTL has expired. Skip while a search is active.
  useFocusEffect(
    useCallback(() => {
      if (searchQuery) return;

      const now = Date.now();
      const ttlExpired =
        lastFetchedAtRef.current === null ||
        now - lastFetchedAtRef.current > FOCUS_REFETCH_TTL_MS;

      if (!ttlExpired) return;

      StorageService.getReadListCache().then((cached) => {
        if (cached && cached.articles.length > 0) {
          setArticles(cached.articles);
          setLoading(false);
          setIsStale(true);
        }
        void load(true);
      });
    }, [searchQuery, load, setArticles, setLoading]),
  );

  const handleArticleArchived = useCallback(
    (articleId: string) => {
      setArticles((prev) => {
        const next = prev.filter((a) => a.id !== articleId);
        void StorageService.saveReadListCache(next);
        return next;
      });
    },
    [setArticles],
  );

  const handleArticlePress = (article: Article) => {
    const currentIndex = articles.findIndex((a) => a.id === article.id);
    navigation.navigate('ArticleDetail', {
      article,
      articles,
      currentIndex,
      onArchived: handleArticleArchived,
    });
  };

  const headerActions = (
    <>
      <IconButton icon="add-outline" onPress={() => setModalVisible(true)} />
      <IconButton icon="search-outline" onPress={() => setSearchVisible(true)} />
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
        onSuccess={handleRefresh}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={search}
      />
    </>
  );
};
