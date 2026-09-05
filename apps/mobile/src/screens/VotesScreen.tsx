import React, { useState, useCallback, useMemo } from 'react';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { ArticleRow } from '../components/common/ArticleRow';
import { IconButton } from '../components/common/IconButton';
import { SearchModal } from '../components/SearchModal';
import { Article, RootStackParamList } from '../types';
import { ExploreService, VotedArticleWithType } from '../services/explore';

type VotesScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Votes'>;

const PAGE_SIZE = 20;

export const VotesScreen: React.FC = () => {
  const navigation = useNavigation<VotesScreenNavigationProp>();

  const [articles, setArticles] = useState<VotedArticleWithType[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [offset, setOffset] = useState(0);
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadVotes = useCallback(async (reset = false) => {
    try {
      const currentOffset = reset ? 0 : offset;

      if (reset) {
        setLoading(true);
        setError(null);
        setOffset(0);
        setHasMore(true);
      }

      const voted = await ExploreService.getUserVotedArticles(PAGE_SIZE, currentOffset);

      if (reset) {
        setArticles(voted);
      } else {
        setArticles((prev) => [...prev, ...voted]);
      }

      setHasMore(voted.length === PAGE_SIZE);
      setOffset(currentOffset + voted.length);
    } catch (err) {
      console.error('Error loading votes:', err);
      if (reset) {
        setError("Couldn't load your votes. Check your connection and try again.");
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
      setLoadingMore(false);
    }
  }, [offset]);

  useFocusEffect(
    useCallback(() => {
      loadVotes(true);
    }, [])
  );

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadVotes(true);
  }, []);

  const handleLoadMore = useCallback(() => {
    if (!loadingMore && hasMore && !loading) {
      setLoadingMore(true);
      loadVotes(false);
    }
  }, [loadingMore, hasMore, loading, loadVotes]);

  const filteredArticles = useMemo((): Article[] => {
    if (!searchQuery) return articles;
    const q = searchQuery.toLowerCase();
    return articles.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        a.description?.toLowerCase().includes(q) ||
        a.author?.toLowerCase().includes(q)
    );
  }, [articles, searchQuery]);

  const handleArticlePress = useCallback((article: Article) => {
    navigation.navigate('ExploreArticleDetail', { article: article as VotedArticleWithType });
  }, [navigation]);

  const renderVoteRow = useCallback((article: Article) => {
    const voted = article as VotedArticleWithType;
    return (
      <ArticleRow
        article={voted}
        onPress={() => handleArticlePress(voted)}
        voteType={voted.voteType}
      />
    );
  }, [handleArticlePress]);

  const clearSearch = useCallback(() => setSearchQuery(null), []);

  const headerActions = (
    <IconButton icon="search-outline" onPress={() => setSearchVisible(true)} accessibilityLabel="Search" />
  );

  return (
    <>
      <ArticleListScreen
        title="Votes"
        articles={filteredArticles}
        loading={loading}
        onBack={() => navigation.goBack()}
        headerActions={headerActions}
        renderItem={renderVoteRow}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        emptyMessage={searchQuery ? 'No matching articles' : 'No voted articles yet'}
        onEndReached={searchQuery ? undefined : handleLoadMore}
        loadingMore={loadingMore}
        searchQuery={searchQuery ?? undefined}
        onClearSearch={clearSearch}
        error={error}
        onRetry={() => loadVotes(true)}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={setSearchQuery}
      />
    </>
  );
};

