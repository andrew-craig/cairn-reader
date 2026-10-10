import React, { ReactNode, useState, useCallback, useRef, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  Alert,
  useColorScheme,
  ActivityIndicator,
  Animated,
  TouchableOpacity,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { ScreenHeader } from './common/ScreenHeader';
import { ListToggle } from './common/ListToggle';
import { Colors, Layout, Spacing, FontSizes, FontFamily, BorderRadius } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { ContentList, SourceRouteType, UnifiedSubscription } from '@cairn/shared';
import { ReadService } from '../services/read';

const AVATAR_SIZE = 48;
const SLIDE_AMOUNT = AVATAR_SIZE + Spacing.lg; // 72px — avatar + gap, so avatar slides off left

/** The `{type}`/`{key}` the list endpoint routes by, or null if the source can't be routed. */
const getRoute = (s: UnifiedSubscription): { type: SourceRouteType; key: string } | null => {
  if (s.type === 'rss' && s.rss_data?.feed_id) return { type: 'rss', key: s.rss_data.feed_id };
  if (s.type === 'email') return { type: 'email', key: s.id };
  return null;
};

interface SourceRowProps {
  title: string;
  subtitle?: string;
  list: ContentList;
  /** Omitted for sources that can't be routed. */
  onListChange?: (list: ContentList) => void;
  onDeletePress?: () => void;
}

const SourceRow: React.FC<SourceRowProps> = ({ title, subtitle, list, onListChange, onDeletePress }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const slideAnim = useRef(new Animated.Value(0)).current;
  const [isOpen, setIsOpen] = useState(false);

  const handlePress = useCallback(() => {
    const toValue = isOpen ? 0 : -SLIDE_AMOUNT;
    Animated.spring(slideAnim, {
      toValue,
      useNativeDriver: true,
      damping: 20,
      stiffness: 300,
    }).start();
    setIsOpen(prev => !prev);
  }, [isOpen, slideAnim]);

  return (
    <View style={[styles.row, { borderColor: colors.border }]}>
      <TouchableOpacity
        style={styles.trashButton}
        onPress={onDeletePress}
        activeOpacity={0.7}
        accessibilityRole="button"
        accessibilityLabel={`Unsubscribe from ${title}`}
      >
        <Ionicons name="trash-outline" size={24} color={colors.error} />
      </TouchableOpacity>
      <Animated.View
        style={[styles.rowContent, { backgroundColor: colors.background, transform: [{ translateX: slideAnim }] }]}
      >
        <TouchableOpacity onPress={handlePress} activeOpacity={1} style={styles.rowInner}>
          <View style={[styles.avatar, { backgroundColor: colors.hover }]}>
            <Ionicons name="person" size={24} color={colors.textSecondary} />
          </View>
          <View style={styles.rowText}>
            <Text style={[styles.rowTitle, { color: colors.text }]} numberOfLines={1}>
              {title}
            </Text>
            {subtitle && (
              <Text style={[styles.rowSubtitle, { color: colors.textSecondary }]} numberOfLines={1}>
                {subtitle}
              </Text>
            )}
            {onListChange && (
              <View style={styles.toggleWrapper}>
                <ListToggle subject={title} list={list} onChange={onListChange} />
              </View>
            )}
          </View>
        </TouchableOpacity>
      </Animated.View>
    </View>
  );
};

interface SubscriptionListScreenProps {
  title: string;
  filter: (s: UnifiedSubscription) => boolean;
  getSubtitle?: (s: UnifiedSubscription) => string | undefined;
  headerActions?: ReactNode;
  emptyMessage?: string;
}

export const SubscriptionListScreen: React.FC<SubscriptionListScreenProps> = ({
  title,
  filter,
  getSubtitle,
  headerActions,
  emptyMessage = 'No subscriptions yet',
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation();

  const [subscriptions, setSubscriptions] = useState<UnifiedSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const subscriptionsRef = useRef(subscriptions);
  useEffect(() => {
    subscriptionsRef.current = subscriptions;
  }, [subscriptions]);

  const load = useCallback(async () => {
    try {
      const response = await ReadService.listAllSubscriptions();
      setSubscriptions(response.subscriptions.filter(filter));
    } catch (error) {
      Alert.alert('Error', 'Failed to load subscriptions. Please try again.');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [filter]);

  useFocusEffect(
    useCallback(() => {
      if (subscriptions.length === 0) setLoading(true);
      load();
    }, [load])
  );

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    load();
  }, [load]);

  const handleUnsubscribe = useCallback((subscription: UnifiedSubscription) => {
    if (subscription.type !== 'rss' || !subscription.rss_data?.feed_id) {
      Alert.alert('Unsupported', 'Unsubscribing from this source type is not yet supported.');
      return;
    }

    const feedId = subscription.rss_data.feed_id;
    Alert.alert(
      'Unsubscribe',
      `Stop receiving new articles from "${subscription.title}"? Articles already in your reading list will be kept.`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Unsubscribe',
          style: 'destructive',
          onPress: async () => {
            const index = subscriptionsRef.current.findIndex(s => s.id === subscription.id);
            setSubscriptions(prev => prev.filter(s => s.id !== subscription.id));
            try {
              await ReadService.unsubscribeFromRSSFeed(feedId);
            } catch (error) {
              setSubscriptions(prev => {
                const next = [...prev];
                next.splice(index, 0, subscription);
                return next;
              });
              Alert.alert('Error', 'Failed to unsubscribe. Please try again.');
            }
          },
        },
      ]
    );
  }, []);

  // Optimistic: flip the toggle now, put it back if the server says no.
  const handleListChange = useCallback(async (subscription: UnifiedSubscription, list: ContentList) => {
    const route = getRoute(subscription);
    if (!route) return;
    const previous = subscription.list;
    const setList = (next: ContentList) =>
      setSubscriptions(prev => prev.map(s => (s.id === subscription.id ? { ...s, list: next } : s)));

    setList(list);
    try {
      await ReadService.setSourceList(route.type, route.key, list);
    } catch (error) {
      console.error('Failed to update source list:', error);
      setList(previous);
      Alert.alert('Error', 'Failed to update. Please try again.');
    }
  }, []);

  const renderItem = useCallback(({ item }: { item: UnifiedSubscription }) => (
    <SourceRow
      title={item.title}
      subtitle={getSubtitle ? getSubtitle(item) : item.description}
      list={item.list}
      onListChange={getRoute(item) ? (list) => handleListChange(item, list) : undefined}
      onDeletePress={() => handleUnsubscribe(item)}
    />
  ), [getSubtitle, handleUnsubscribe, handleListChange]);

  const keyExtractor = useCallback((item: UnifiedSubscription) => item.id, []);

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <FlatList
        data={subscriptions}
        renderItem={renderItem}
        keyExtractor={keyExtractor}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        ListHeaderComponent={
          <>
            <ScreenHeader title={title} onBack={navigation.canGoBack() ? () => navigation.goBack() : undefined} rightActions={headerActions} />
            {subscriptions.some(s => getRoute(s)) && (
              <Text style={[styles.caption, { color: colors.textSecondary }]}>
                Choose where each source's new items land. This applies to new items only; items already delivered stay where they are.
              </Text>
            )}
          </>
        }
        ListEmptyComponent={
          loading ? (
            <View style={styles.centered}>
              <ActivityIndicator size="large" color={colors.primary} />
            </View>
          ) : (
            <View style={GlobalStyles.emptyContainer}>
              <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
                {emptyMessage}
              </Text>
            </View>
          )
        }
        contentContainerStyle={[
          styles.listContent,
          { paddingBottom: Layout.tabBarHeight + insets.bottom + Spacing.lg },
        ]}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  listContent: {
    flexGrow: 1,
  },
  centered: {
    padding: Spacing.xxl,
    alignItems: 'center',
    justifyContent: 'center',
  },
  row: {
    position: 'relative',
    borderBottomWidth: 1,
    overflow: 'hidden',
  },
  trashButton: {
    position: 'absolute',
    right: 0,
    top: 0,
    bottom: 0,
    width: SLIDE_AMOUNT,
    alignItems: 'center',
    justifyContent: 'center',
    paddingRight: Spacing.md,
  },
  rowContent: {},
  rowInner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.lg,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.lg,
  },
  avatar: {
    width: AVATAR_SIZE,
    height: AVATAR_SIZE,
    borderRadius: BorderRadius.md,
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  rowText: {
    flex: 1,
    gap: Spacing.xs,
  },
  toggleWrapper: {
    marginTop: Spacing.xs,
  },
  caption: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    paddingHorizontal: Spacing.md,
    paddingBottom: Spacing.md,
  },
  rowTitle: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultBold,
  },
  rowSubtitle: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
  },
});
