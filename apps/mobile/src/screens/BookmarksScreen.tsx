import React, { useCallback, useState } from 'react';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { SearchModal } from '../components/SearchModal';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';
import { ArticleStore } from '../services/articleStore';
import { useCursorArticleList, PAGE_SIZE } from '../hooks/useCursorArticleList';

type BookmarksScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Bookmarks'>;

export const BookmarksScreen: React.FC = () => {
  const navigation = useNavigation<BookmarksScreenNavigationProp>();
  const [searchVisible, setSearchVisible] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPage = useCallback(
    (cursor: string | undefined) =>
      ReadService.listUserContents({ is_favorite: true, limit: PAGE_SIZE, cursor }),
    [],
  );

  const onResetLoaded = useCallback((next: Article[]) => {
    void ArticleStore.upsertMany(next);
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
    searchQuery,
    load,
    search,
    clearSearch,
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

  const headerActions = (
    <IconButton icon="search-outline" onPress={() => setSearchVisible(true)} accessibilityLabel="Search" />
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
        error={error}
        onRetry={() => load(true)}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={search}
      />
    </>
  );
};
