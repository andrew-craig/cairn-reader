import React, { ReactNode, useState, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  Alert,
  useColorScheme,
  ActivityIndicator,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { ScreenHeader } from './common/ScreenHeader';
import { Colors, Layout, Spacing, FontSizes, FontFamily, BorderRadius } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { UnifiedSubscription } from '../types/read';
import { ReadService } from '../services/read';

interface SourceRowProps {
  title: string;
  subtitle?: string;
}

const SourceRow: React.FC<SourceRowProps> = ({ title, subtitle }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  return (
    <View style={[styles.row, { borderColor: colors.border }]}>
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
      </View>
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

  const renderItem = useCallback(({ item }: { item: UnifiedSubscription }) => (
    <SourceRow
      title={item.title}
      subtitle={getSubtitle ? getSubtitle(item) : item.description}
    />
  ), [getSubtitle]);

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
          <ScreenHeader title={title} onBack={navigation.canGoBack() ? () => navigation.goBack() : undefined} rightActions={headerActions} />
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
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.lg,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.lg,
    borderBottomWidth: 1,
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: BorderRadius.md,
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  rowText: {
    flex: 1,
    gap: Spacing.xs,
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
