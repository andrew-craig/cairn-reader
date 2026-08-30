import React, { useCallback, useState } from 'react';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { SearchModal } from '../components/SearchModal';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';
import { useCursorArticleList, PAGE_SIZE } from '../hooks/useCursorArticleList';

type BookmarksScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Bookmarks'>;

export const BookmarksScreen: React.FC = () => {
  const navigation = useNavigation<BookmarksScreenNavigationProp>();
  const [searchVisible, setSearchVisible] = useState(false);

  const fetchPage = useCallback(
    (cursor: string | undefined) =>
      ReadService.listUserContents({ is_favorite: true, limit: PAGE_SIZE, cursor }),
    [],
  );

  const {
    articles,
    loading,
    refreshing,
    loadingMore,
    searchQuery,
    load,
    search,
    clearSearch,
    handleRefresh,
    handleLoadMore,
  } = useCursorArticleList({ fetchPage });

  useFocusEffect(
    useCallback(() => {
      if (!searchQuery) {
        void load(true);
      }
    }, [searchQuery, load]),
  );

  const handleArticlePress = (article: Article) => {
    navigation.navigate('ArticleDetail', { article });
  };

  const headerActions = (
    <IconButton icon="search-outline" onPress={() => setSearchVisible(true)} />
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
        onSearch={search}
      />
    </>
  );
};
