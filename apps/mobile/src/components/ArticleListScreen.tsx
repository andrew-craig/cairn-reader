import React, { ReactElement, ReactNode, useRef } from 'react';
import {
  View,
  Text,
  useColorScheme,
  FlatList,
  ActivityIndicator,
  TouchableOpacity,
  ViewToken,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, GlobalStyles, Layout, Spacing, FontSizes, FontFamily } from '../constants';
import { ArticleRow } from './common/ArticleRow';
import { ScreenHeader } from './common/ScreenHeader';
import { Article } from '../types';

interface ArticleListScreenProps {
  title: string;
  articles: Article[];
  loading: boolean;
  onBack?: () => void;
  headerActions?: ReactNode;
  onArticlePress?: (article: Article) => void;
  renderItem?: (article: Article) => ReactElement;
  onRefresh?: () => void;
  refreshing?: boolean;
  emptyMessage?: string;
  onEndReached?: () => void;
  onViewableItemsChanged?: (info: {
    viewableItems: ViewToken[];
    changed: ViewToken[];
  }) => void;
  onScrollBeginDrag?: () => void;
  loadingMore?: boolean;
  endOfListMessage?: string;
  searchQuery?: string;
  onClearSearch?: () => void;
}

export const ArticleListScreen: React.FC<ArticleListScreenProps> = ({
  title,
  articles,
  loading,
  onBack,
  headerActions,
  onArticlePress,
  renderItem: renderItemProp,
  onRefresh,
  refreshing = false,
  emptyMessage = 'No articles found',
  onEndReached,
  onViewableItemsChanged,
  onScrollBeginDrag,
  loadingMore = false,
  endOfListMessage,
  searchQuery,
  onClearSearch,
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
    <View>
      <ScreenHeader title={title} onBack={onBack} rightActions={headerActions} />
      {searchQuery && onClearSearch && (
        <View style={[searchBannerStyles.container, { backgroundColor: colors.hover }]}>
          <Text style={[searchBannerStyles.text, { color: colors.textSecondary }]} numberOfLines={1}>
            Results for "{searchQuery}"
          </Text>
          <TouchableOpacity onPress={onClearSearch} hitSlop={8}>
            <Ionicons name="close-circle" size={18} color={colors.textSecondary} />
          </TouchableOpacity>
        </View>
      )}
    </View>
  );

  const renderArticle = ({ item }: { item: Article }) => {
    if (renderItemProp) return renderItemProp(item);
    return <ArticleRow article={item} onPress={() => onArticlePress?.(item)} />;
  };

  const renderEmpty = () => (
    <View style={GlobalStyles.emptyContainer}>
      <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
        {emptyMessage}
      </Text>
    </View>
  );

  const renderFooter = () => {
    if (loadingMore) {
      return (
        <View style={{ padding: 20, alignItems: 'center' }}>
          <ActivityIndicator size="small" color={colors.primary} />
        </View>
      );
    }

    if (endOfListMessage && articles.length > 0) {
      return (
        <View style={{ padding: 20, alignItems: 'center' }}>
          <Text
            style={{
              color: colors.textSecondary,
              fontSize: FontSizes.sm,
              fontFamily: FontFamily.default,
              textAlign: 'center',
            }}
          >
            {endOfListMessage}
          </Text>
        </View>
      );
    }

    return null;
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
        onScrollBeginDrag={onScrollBeginDrag}
        viewabilityConfig={viewabilityConfigRef.current}
        contentContainerStyle={{ paddingBottom: bottomPadding }}
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
