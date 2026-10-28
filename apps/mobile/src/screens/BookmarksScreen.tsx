import React, { useCallback, useState } from 'react';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';
import { ArticleStore } from '../services/articleStore';
import { useCursorArticleList, PAGE_SIZE } from '../hooks/useCursorArticleList';

type BookmarksScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Bookmarks'>;

export const BookmarksScreen: React.FC = () => {
  const navigation = useNavigation<BookmarksScreenNavigationProp>();
  const [error, setError] = useState<string | null>(null);

  const fetchPage = useCallback(
    (cursor: string | undefined) =>
      ReadService.listUserContents({ is_favorite: true, limit: PAGE_SIZE, cursor }),
    [],
  );

  const onResetLoaded = useCallback((next: Article[]) => {
    // Favorited Feed items are online-only: keep them out of the offline
    // Reads store (they'd show up in Reads and get prefetched).
    void ArticleStore.upsertMany(next.filter((a) => a.list === 'reads'));
    setError(null);
  }, []);

  const onLoadError = useCallback((reset: boolean) => {
    if (reset) {
      setError("Couldn't load your bookmarks. Check your connection and try again.");
    }
  }, []);

  const {
    articles,
    setArticles,
    loading,
    setLoading,
    refreshing,
    loadingMore,
    searchOpen,
    query,
    setQuery,
    searchQuery,
    openSearch,
    closeSearch,
    load,
    handleRefresh,
    handleLoadMore,
  } = useCursorArticleList({ fetchPage, onResetLoaded, onLoadError });

  // Show stored favorites immediately, then refresh from the network — a
  // network failure at that point no longer blanks the list, since the store
  // already primed `articles`.
  useFocusEffect(
    useCallback(() => {
      if (searchQuery) return;

      ArticleStore.listFavorites().then((stored) => {
        if (stored.length > 0) {
          setArticles(stored);
          setLoading(false);
        }
        void load(true);
      });
    }, [searchQuery, load, setArticles, setLoading]),
  );

  const handleArticlePress = (article: Article) => {
    navigation.navigate('ArticleDetail', { article });
  };

  return (
    <ArticleListScreen
      title="Bookmarks"
      articles={articles}
      loading={loading}
      onBack={() => navigation.goBack()}
      onArticlePress={handleArticlePress}
      onRefresh={handleRefresh}
      refreshing={refreshing}
      emptyMessage={searchQuery ? 'No matching articles' : 'No bookmarked articles yet'}
      onEndReached={handleLoadMore}
      loadingMore={loadingMore}
      search={{
        open: searchOpen,
        query,
        onChangeQuery: setQuery,
        onOpen: openSearch,
        onClose: closeSearch,
        placeholder: 'Search your bookmarks',
      }}
      error={error}
      onRetry={() => load(true)}
    />
  );
};
