import React, { useState, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  useColorScheme,
  ActivityIndicator,
  Modal,
  Platform,
} from 'react-native';
import { BlurView } from 'expo-blur';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { IconButton } from '../components/common/IconButton';
import { Colors, Spacing, FontSizes, FontFamily, BorderRadius } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { RootStackParamList } from '../types';
import { UnifiedSubscription } from '../types/read';
import { ReadService } from '../services/read';

type NewslettersScreenNavigationProp = StackNavigationProp<RootStackParamList, 'Newsletters'>;

interface SubscriptionRowProps {
  subscription: UnifiedSubscription;
  isDark: boolean;
}

const SubscriptionRow: React.FC<SubscriptionRowProps> = ({ subscription, isDark }) => {
  const colors = isDark ? Colors.dark : Colors.light;
  const email = subscription.email_data?.email_address;

  return (
    <View style={[styles.row, { borderColor: colors.border }]}>
      <View style={[styles.avatar, { backgroundColor: colors.hover }]}>
        <Text style={[styles.avatarText, { color: colors.textSecondary }]}>
          {subscription.title.charAt(0).toUpperCase()}
        </Text>
      </View>
      <View style={styles.rowContent}>
        <Text style={[styles.rowTitle, { color: colors.text }]} numberOfLines={1}>
          {subscription.title}
        </Text>
        {email && (
          <Text style={[styles.rowSubtitle, { color: colors.textSecondary }]} numberOfLines={1}>
            {email}
          </Text>
        )}
      </View>
    </View>
  );
};

interface AddNewsletterModalProps {
  visible: boolean;
  onClose: () => void;
}

const AddNewsletterModal: React.FC<AddNewsletterModalProps> = ({ visible, onClose }) => {
  const colors = Colors.light;

  return (
    <Modal visible={visible} transparent animationType="fade">
      <BlurView
        style={styles.modalOverlay}
        intensity={Platform.OS === 'ios' ? 8 : 16}
        tint="light"
      >
        <View
          style={[
            styles.modalContent,
            { backgroundColor: colors.hover },
          ]}
        >
          <Text style={[styles.modalLabel, { color: colors.text }]}>
            Send or forward emails to
          </Text>
          <Text style={[styles.modalEmail, { color: colors.text }]}>
            rose-parrot-74@cairnreader.app
          </Text>
          <Text style={[styles.modalDescription, { color: colors.textSecondary }]}>
            New subscriptions will be automatically added to your reading list.
          </Text>
        </View>
      </BlurView>
    </Modal>
  );
};

export const NewslettersScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const colors = isDark ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<NewslettersScreenNavigationProp>();

  const [subscriptions, setSubscriptions] = useState<UnifiedSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);

  const loadSubscriptions = useCallback(async () => {
    try {
      const response = await ReadService.listAllSubscriptions();
      setSubscriptions(response.subscriptions.filter(s => s.type === 'email'));
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
    <>
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
                styles.header,
                {
                  backgroundColor: colors.background,
                  paddingTop: insets.top + Spacing.md,
                  borderColor: colors.border,
                },
              ]}
            >
              <View style={styles.headerLeft}>
                <IconButton icon="chevron-back" onPress={() => navigation.goBack()} />
                <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>
                  Newsletters
                </Text>
              </View>
              <View style={styles.headerRight}>
                <IconButton
                  icon="add"
                  onPress={() => setModalVisible(true)}
                  size={24}
                />
                <IconButton
                  icon="search"
                  onPress={() => {}}
                  size={24}
                />
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
                  No newsletters yet
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
      <AddNewsletterModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
      />
    </>
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
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    borderBottomWidth: 1,
    height: undefined,
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm,
    flex: 1,
  },
  headerRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.sm,
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
  },
  avatarText: {
    fontSize: FontSizes.lg,
    fontFamily: FontFamily.defaultMedium,
  },
  rowContent: {
    flex: 1,
    gap: Spacing.xs,
  },
  rowTitle: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultMedium,
  },
  rowSubtitle: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
  },
  modalOverlay: {
    flex: 1,
    justifyContent: 'flex-start',
    alignItems: 'center',
    paddingTop: 80,
  },
  modalContent: {
    marginHorizontal: Spacing.md,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    borderRadius: 21,
    gap: Spacing.md,
    alignItems: 'center',
  },
  modalLabel: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    textAlign: 'center',
  },
  modalEmail: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.defaultBold,
    textAlign: 'center',
  },
  modalDescription: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    textAlign: 'center',
  },
});
