import React, { useState, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  useColorScheme,
  ActivityIndicator,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { IconButton } from '../components/common/IconButton';
import { Colors, Spacing, FontSizes, FontFamily, BorderRadius } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { RootStackParamList } from '../types';
import { UnifiedSubscription } from '../types/read';
import { ReadService } from '../services/read';

type FeedsScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Feeds'>;

const SubscriptionTypeLabel: Record<string, string> = {
  rss: 'RSS',
  social: 'Social',
  email: 'Email',
};

interface SubscriptionRowProps {
  subscription: UnifiedSubscription;
  isDark: boolean;
}

const SubscriptionRow: React.FC<SubscriptionRowProps> = ({ subscription, isDark }) => {
  const colors = isDark ? Colors.dark : Colors.light;

  const subtitle = subscription.type === 'rss' && subscription.rss_data?.feed_url
    ? subscription.rss_data.feed_url
    : subscription.type === 'email' && subscription.email_data?.email_address
      ? subscription.email_data.email_address
      : subscription.description;

  return (
    <View style={[styles.row, { borderColor: colors.border }]}>
      <View style={styles.rowContent}>
        <View style={styles.rowHeader}>
          <Text style={[styles.rowTitle, { color: colors.text }]} numberOfLines={1}>
            {subscription.title}
          </Text>
          <View style={[styles.typeBadge, { backgroundColor: colors.hover }]}>
            <Text style={[styles.typeBadgeText, { color: colors.textSecondary }]}>
              {SubscriptionTypeLabel[subscription.type] || subscription.type}
            </Text>
          </View>
        </View>
        {subtitle && (
          <Text style={[styles.rowSubtitle, { color: colors.textSecondary }]} numberOfLines={1}>
            {subtitle}
          </Text>
        )}
      </View>
    </View>
  );
};

export const FeedsScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const colors = isDark ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<FeedsScreenNavigationProp>();

  const [subscriptions, setSubscriptions] = useState<UnifiedSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const loadSubscriptions = useCallback(async () => {
    try {
      const response = await ReadService.listAllSubscriptions();
      setSubscriptions(response.subscriptions);
    } catch (error) {
      console.error('Error loading subscriptions:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      setLoading(true);
      loadSubscriptions();
    }, [loadSubscriptions])
  );

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadSubscriptions();
  }, [loadSubscriptions]);

  const renderItem = useCallback(({ item }: { item: UnifiedSubscription }) => (
    <SubscriptionRow subscription={item} isDark={isDark} />
  ), [isDark]);

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
              <IconButton icon="chevron-back" onPress={() => navigation.goBack()} />
              <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>Feeds</Text>
            </View>
          </View>
        }
        ListEmptyComponent={
          loading ? (
            <View style={styles.centered}>
              <ActivityIndicator size="large" color={colors.primary} />
            </View>
          ) : (
            <View style={GlobalStyles.emptyContainer}>
              <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
                No subscriptions yet
              </Text>
            </View>
          )
        }
        contentContainerStyle={[
          styles.listContent,
          { paddingBottom: insets.bottom + Spacing.lg },
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
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    borderBottomWidth: 1,
  },
  rowContent: {
    gap: Spacing.xs,
  },
  rowHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm,
  },
  rowTitle: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultMedium,
    flex: 1,
  },
  typeBadge: {
    paddingHorizontal: Spacing.sm,
    paddingVertical: 2,
    borderRadius: BorderRadius.sm,
  },
  typeBadgeText: {
    fontSize: FontSizes.xs,
    fontFamily: FontFamily.defaultMedium,
  },
  rowSubtitle: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
  },
});
