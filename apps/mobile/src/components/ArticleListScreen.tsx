import React, { ReactNode, useRef } from 'react';
import {
  View,
  Text,
  useColorScheme,
  FlatList,
  ActivityIndicator,
  ViewToken,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, GlobalStyles, Layout, Spacing } from '../constants';
import { ArticleRow } from './common/ArticleRow';
import { Article } from '../types';

interface ArticleListScreenProps {
  title: string;
  articles: Article[];
  loading: boolean;
  headerActions?: ReactNode;
  onArticlePress: (article: Article) => void;
  onRefresh?: () => void;
  refreshing?: boolean;
  emptyMessage?: string;
  onEndReached?: () => void;
  onViewableItemsChanged?: (info: {
    viewableItems: ViewToken[];
    changed: ViewToken[];
  }) => void;
  loadingMore?: boolean;
}

export const ArticleListScreen: React.FC<ArticleListScreenProps> = ({
  title,
  articles,
  loading,
  headerActions,
  onArticlePress,
  onRefresh,
  refreshing = false,
  emptyMessage = 'No articles found',
  onEndReached,
  onViewableItemsChanged,
  loadingMore = false,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  // Calculate bottom padding to account for tab bar + bottom safe area
  const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;

  // Memoize viewability config to prevent recreating on each render
  const viewabilityConfigRef = useRef({
    itemVisiblePercentThreshold: 50,
  });

  const renderHeader = () => (
    <View
      style={[
        GlobalStyles.header,
        {
          backgroundColor: colors.background,
          paddingTop: insets.top + Spacing.md,
          height: undefined, // Override fixed height to allow safe area padding
        },
      ]}
    >
      <View style={GlobalStyles.headerLeft}>
        <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>{title}</Text>
      </View>
      {headerActions && (
        <View style={GlobalStyles.headerRight}>{headerActions}</View>
      )}
    </View>
  );

  const renderArticle = ({ item }: { item: Article }) => (
    <ArticleRow article={item} onPress={() => onArticlePress(item)} />
  );

  const renderEmpty = () => (
    <View style={GlobalStyles.emptyContainer}>
      <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
        {emptyMessage}
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
        onRefresh={onRefresh}
        refreshing={refreshing}
        onEndReached={onEndReached}
        onEndReachedThreshold={0.5}
        onViewableItemsChanged={onViewableItemsChanged}
        viewabilityConfig={viewabilityConfigRef.current}
        contentContainerStyle={{ paddingBottom: bottomPadding }}
      />
    </View>
  );
};
