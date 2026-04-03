import React, { useState, useCallback, useMemo } from 'react';
import {
  View,
  Text,
  FlatList,
  ActivityIndicator,
  useColorScheme,
  TouchableOpacity,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, GlobalStyles, Layout, Spacing, FontSizes, FontFamily } from '../constants';
import { ArticleRow } from '../components/common/ArticleRow';
import { IconButton } from '../components/common/IconButton';
import { SearchModal } from '../components/SearchModal';
import { RootStackParamList } from '../types';
import { ExploreService, VotedArticleWithType } from '../services/explore';

type VotesScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Votes'>;

const PAGE_SIZE = 20;

export const VotesScreen: React.FC = () => {
  const navigation = useNavigation<VotesScreenNavigationProp>();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  const [articles, setArticles] = useState<VotedArticleWithType[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [offset, setOffset] = useState(0);
  const [searchVisible, setSearchVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string | null>(null);

  const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;

  const loadVotes = useCallback(async (reset = false) => {
    try {
      const currentOffset = reset ? 0 : offset;

      if (reset) {
        setLoading(true);
        setOffset(0);
        setHasMore(true);
      }

      const votedArticles = await ExploreService.getUserVotedArticles(
        PAGE_SIZE,
        currentOffset
      );

      if (reset) {
        setArticles(votedArticles);
      } else {
        setArticles((prev) => [...prev, ...votedArticles]);
      }

      setHasMore(votedArticles.length === PAGE_SIZE);
      setOffset(currentOffset + votedArticles.length);
    } catch (error) {
      console.error('Error loading votes:', error);
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

  const handleArticlePress = (article: VotedArticleWithType) => {
    navigation.navigate('ExploreArticleDetail', { article });
  };

  const handleSearchPress = () => {
    setSearchVisible(true);
  };

  const handleSearch = (query: string) => {
    setSearchQuery(query);
  };

  const clearSearch = () => {
    setSearchQuery(null);
  };

  const filteredArticles = useMemo(() => {
    if (!searchQuery) return articles;
    const q = searchQuery.toLowerCase();
    return articles.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        a.description?.toLowerCase().includes(q) ||
        a.author?.toLowerCase().includes(q)
    );
  }, [articles, searchQuery]);

  const renderHeader = () => (
    <View>
      <View
        style={[
          GlobalStyles.header,
          {
            backgroundColor: colors.background,
            paddingTop: insets.top + Spacing.md,
            height: undefined,
          },
        ]}
      >
        <View style={GlobalStyles.headerLeft}>
          <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>Votes</Text>
        </View>
        <View style={GlobalStyles.headerRight}>
          <IconButton icon="search-outline" onPress={handleSearchPress} />
        </View>
      </View>
      {searchQuery && (
        <View style={[searchBannerStyles.container, { backgroundColor: colors.hover }]}>
          <Text style={[searchBannerStyles.text, { color: colors.textSecondary }]} numberOfLines={1}>
            Results for "{searchQuery}"
          </Text>
          <TouchableOpacity onPress={clearSearch} hitSlop={8}>
            <Ionicons name="close-circle" size={18} color={colors.textSecondary} />
          </TouchableOpacity>
        </View>
      )}
    </View>
  );

  const renderArticle = ({ item }: { item: VotedArticleWithType }) => (
    <ArticleRow
      article={item}
      onPress={() => handleArticlePress(item)}
      voteType={item.voteType}
    />
  );

  const renderEmpty = () => (
    <View style={GlobalStyles.emptyContainer}>
      <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
        {searchQuery ? 'No matching articles' : 'No voted articles yet'}
      </Text>
    </View>
  );

  const renderFooter = () => {
    if (!loadingMore) return null;

    return (
      <View style={{ padding: 20, alignItems: 'center' }}>
        <ActivityIndicator size="small" color={colors.primary} />
      </View>
    );
  };

  if (loading) {
    return (
      <View
        style={[
          GlobalStyles.container,
          GlobalStyles.centerContent,
          { backgroundColor: colors.background },
        ]}
      >
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  return (
    <View style={[GlobalStyles.container, { backgroundColor: colors.background }]}>
      <FlatList
        data={filteredArticles}
        renderItem={renderArticle}
        keyExtractor={(item) => item.id}
        ListHeaderComponent={renderHeader}
        ListEmptyComponent={renderEmpty}
        ListFooterComponent={renderFooter}
        showsVerticalScrollIndicator={false}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        onEndReached={searchQuery ? undefined : handleLoadMore}
        onEndReachedThreshold={0.5}
        contentContainerStyle={{ paddingBottom: bottomPadding }}
      />
      <SearchModal
        visible={searchVisible}
        onClose={() => setSearchVisible(false)}
        onSearch={handleSearch}
      />
    </View>
  );
};

const searchBannerStyles = {
  container: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    justifyContent: 'space-between' as const,
    marginHorizontal: Spacing.md,
    marginBottom: Spacing.sm,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.sm,
    borderRadius: 8,
  },
  text: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.defaultMedium,
    flex: 1,
    marginRight: Spacing.sm,
  },
};
