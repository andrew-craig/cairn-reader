import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, useColorScheme, ActivityIndicator } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useAuth } from '../contexts/AuthContext';
import { Colors, Spacing, FontSizes, FontFamily, Layout } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { ChevronRightIcon } from '../components/icons';
import { RootStackParamList } from '../types';
import { ExploreService } from '../services/explore';
import { ReadService } from '../services/read';
import { pluralize } from '../utils/helpers';

type YouScreenNavigationProp = StackNavigationProp<RootStackParamList>;

interface MenuItemProps {
  title: string;
  subtitle?: string;
  onPress: () => void;
  isDark: boolean;
}

const MenuItem: React.FC<MenuItemProps> = ({ title, subtitle, onPress, isDark }) => {
  const colors = isDark ? Colors.dark : Colors.light;

  return (
    <TouchableOpacity
      style={[styles.menuItem, { borderColor: colors.border }]}
      onPress={onPress}
      activeOpacity={0.7}
    >
      <View style={styles.menuItemContent}>
        <Text style={[styles.menuItemTitle, { color: colors.text }]}>{title}</Text>
        {subtitle && (
          <Text style={[styles.menuItemSubtitle, { color: colors.textSecondary }]}>
            {subtitle}
          </Text>
        )}
      </View>
      <ChevronRightIcon size={24} color={colors.textSecondary} />
    </TouchableOpacity>
  );
};

interface SpacerProps {
  isDark: boolean;
}

const Spacer: React.FC<SpacerProps> = ({ isDark }) => {
  const colors = isDark ? Colors.dark : Colors.light;
  return <View style={[styles.spacer, { borderColor: colors.border }]} />;
};

export const YouScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const { user, logout } = useAuth();
  const colors = isDark ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<YouScreenNavigationProp>();

  // Calculate bottom padding to account for tab bar + bottom safe area
  const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;

  // State for user statistics
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [feedsCount, setFeedsCount] = useState(0);
  const [newslettersCount, setNewslettersCount] = useState(0);
  const [bookmarksCount, setBookmarksCount] = useState(0);
  const [upVotesCount, setUpVotesCount] = useState(0);
  const [downVotesCount, setDownVotesCount] = useState(0);

  const accountName = user?.email || 'Anonymous user';

  // Fetch user statistics every time the screen comes into focus so that
  // changes made elsewhere (e.g. voting) are reflected on return.
  useFocusEffect(
    useCallback(() => {
      const fetchStats = async () => {
        setLoading(true);
        setError(null);

        // Fetch all stats in parallel, tolerating individual failures so that
        // a broken votes endpoint doesn't hide feed/bookmark counts.
        const [voteResult, subscriptionsResult, bookmarksResult] = await Promise.allSettled([
          ExploreService.getUserVoteStats(),
          ReadService.listAllSubscriptions(),
          ReadService.listUserContents({ limit: 1 }),
        ]);

        if (voteResult.status === 'fulfilled') {
          setUpVotesCount(voteResult.value.upvotes);
          setDownVotesCount(voteResult.value.downvotes);
        } else {
          console.error('Error fetching vote stats:', voteResult.reason);
          setError(voteResult.reason instanceof Error ? voteResult.reason.message : 'Failed to get voted articles');
        }

        if (subscriptionsResult.status === 'fulfilled') {
          const subs = subscriptionsResult.value.subscriptions;
          setFeedsCount(subs.filter(s => s.type !== 'email').length);
          setNewslettersCount(subs.filter(s => s.type === 'email').length);
        } else {
          console.error('Error fetching subscriptions:', subscriptionsResult.reason);
        }

        if (bookmarksResult.status === 'fulfilled') {
          setBookmarksCount(bookmarksResult.value.total_count);
        } else {
          console.error('Error fetching bookmarks:', bookmarksResult.reason);
        }

        setLoading(false);
      };

      fetchStats();
    }, [])
  );

  const handleAccountPress = () => {
    navigation.navigate('Account');
  };

  const handleFeedsPress = () => {
    navigation.navigate('Feeds');
  };

  const handleNewslettersPress = () => {
    navigation.navigate('Newsletters');
  };

  const handleBookmarksPress = () => {
    navigation.navigate('Bookmarks');
  };

  const handleVotesPress = () => {
    navigation.navigate('Votes');
  };

  const handleLogoutPress = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Error logging out:', error);
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={[styles.scrollContent, { paddingBottom: bottomPadding }]}
      >
        <View
          style={[
            styles.header,
            {
              backgroundColor: colors.background,
              paddingTop: insets.top + Spacing.md,
              height: undefined, // Override fixed height to allow safe area padding
            },
          ]}
        >
          <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>You</Text>
        </View>

        {error && (
          <View style={[styles.errorContainer, { backgroundColor: colors.border }]}>
            <Text style={[styles.errorText, { color: colors.error }]}>
              {error}
            </Text>
          </View>
        )}

        <View style={styles.menuSection}>
          <MenuItem
            title="Account"
            subtitle={accountName}
            onPress={handleAccountPress}
            isDark={isDark}
          />
          <Spacer isDark={isDark} />
          <MenuItem
            title="Feeds"
            subtitle={loading ? 'Loading...' : `${feedsCount} ${pluralize(feedsCount, 'subscription')}`}
            onPress={handleFeedsPress}
            isDark={isDark}
          />
          <MenuItem
            title="Newsletters"
            subtitle={loading ? 'Loading...' : `${newslettersCount} ${pluralize(newslettersCount, 'subscription')}`}
            onPress={handleNewslettersPress}
            isDark={isDark}
          />
          <MenuItem
            title="Bookmarks"
            subtitle={loading ? 'Loading...' : `${bookmarksCount} saved`}
            onPress={handleBookmarksPress}
            isDark={isDark}
          />
          <MenuItem
            title="Votes"
            subtitle={
              loading
                ? 'Loading...'
                : `${upVotesCount} up ${pluralize(upVotesCount, 'vote')}, ${downVotesCount} down ${pluralize(downVotesCount, 'vote')}`
            }
            onPress={handleVotesPress}
            isDark={isDark}
          />
          <Spacer isDark={isDark} />
          <MenuItem
            title="Log out"
            onPress={handleLogoutPress}
            isDark={isDark}
          />
        </View>
      </ScrollView>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
  },
  header: {
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    height: 48,
    justifyContent: 'center',
  },
  errorContainer: {
    marginHorizontal: Spacing.md,
    marginTop: Spacing.md,
    padding: Spacing.md,
    borderRadius: 8,
  },
  errorText: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    textAlign: 'center',
  },
  menuSection: {
    marginTop: Spacing.md,
  },
  menuItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    borderTopWidth: 1,
    borderBottomWidth: 1,
  },
  menuItemContent: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.md,
    marginRight: Spacing.sm,
  },
  menuItemTitle: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  menuItemSubtitle: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  spacer: {
    height: Spacing.xl,
    borderTopWidth: 1,
    borderBottomWidth: 1,
  },
});
