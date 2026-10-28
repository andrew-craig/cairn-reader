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
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, GlobalStyles, Layout, Spacing, FontSizes, FontFamily } from '../constants';
import { ArticleRow } from './common/ArticleRow';
import { ScreenHeader } from './common/ScreenHeader';
import { HeaderSearchProps } from './common/HeaderSearch';
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
  /** Header search. While it is open the screen keeps its header mounted. */
  search?: HeaderSearchProps;
  /** When set, renders a small banner below the header to indicate stale data. */
  staleMessage?: string;
  /**
   * When set and there are no articles to show, renders a full-screen error
   * state with an explicit retry affordance instead of the empty message — so a
   * network failure is not mistaken for an empty account.
   */
  error?: string | null;
  onRetry?: () => void;
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
  search,
  staleMessage,
  error,
  onRetry,
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

  const renderHeader = (showStaleMessage = true) => (
    <View>
      <ScreenHeader title={title} onBack={onBack} rightActions={headerActions} search={search} />
      {showStaleMessage && staleMessage && !search?.open && (
        <View style={[searchBannerStyles.container, { backgroundColor: colors.hover }]}>
          <Text style={[searchBannerStyles.text, { color: colors.textSecondary }]} numberOfLines={1}>
            {staleMessage}
          </Text>
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

  // Only take over the whole screen with a spinner on the first load. Once
  // articles are on screen, a reload (pull-to-refresh, search) keeps the list
  // visible and shows progress via the RefreshControl instead of replacing the
  // populated list with a spinner. Never while the search field is open: that
  // would unmount it mid-typing.
  if (loading && articles.length === 0 && !search?.open) {
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

  // A load failure with nothing to show: distinguish it from an empty account
  // and give an explicit retry (pull-to-refresh alone is an accessibility gap).
  // Suppressed while the search field is open — a failed reset load shouldn't
  // hide empty search results behind a stale Retry that would abandon the search.
  if (error && articles.length === 0 && !search?.open) {
    return (
      <View style={[GlobalStyles.container, { backgroundColor: colors.background }]}>
        {renderHeader(false)}
        <View style={GlobalStyles.emptyContainer}>
          <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>{error}</Text>
          {onRetry && (
            <TouchableOpacity
              onPress={onRetry}
              style={[errorStateStyles.retryButton, { borderColor: colors.textSecondary }]}
              accessibilityRole="button"
              accessibilityLabel="Retry loading"
            >
              <Text style={[errorStateStyles.retryText, { color: colors.text }]}>Retry</Text>
            </TouchableOpacity>
          )}
        </View>
      </View>
    );
  }

  return (
    <View style={[GlobalStyles.container, { backgroundColor: colors.background }]}>
      <FlatList
        data={articles}
        renderItem={renderArticle}
        keyExtractor={(item) => item.id}
        // An element, not a component: a new component type each render would
        // remount the header and drop focus from the search field.
        ListHeaderComponent={renderHeader()}
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

const errorStateStyles = {
  retryButton: {
    marginTop: Spacing.md,
    paddingVertical: Spacing.sm,
    paddingHorizontal: Spacing.lg,
    borderWidth: 1,
    borderRadius: 8,
  },
  retryText: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.defaultSemiBold,
  },
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
