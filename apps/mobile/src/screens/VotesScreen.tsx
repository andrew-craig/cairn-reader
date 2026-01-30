import React, { useState, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  ActivityIndicator,
  useColorScheme,
  TouchableOpacity,
} from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, GlobalStyles, Layout, Spacing } from '../constants';
import { ArticleRow } from '../components/common/ArticleRow';
import { IconButton } from '../components/common/IconButton';
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
    console.log('Search pressed');
  };

  const renderHeader = () => (
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
        No voted articles yet
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
        data={articles}
        renderItem={renderArticle}
        keyExtractor={(item) => item.id}
        ListHeaderComponent={renderHeader}
        ListEmptyComponent={renderEmpty}
        ListFooterComponent={renderFooter}
        showsVerticalScrollIndicator={false}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.5}
        contentContainerStyle={{ paddingBottom: bottomPadding }}
      />
    </View>
  );
};
