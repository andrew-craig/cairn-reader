import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, useColorScheme, ActivityIndicator } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { useAuth } from '../contexts/AuthContext';
import { Colors, Spacing, FontSizes, FontFamily, Layout } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { ChevronRightIcon } from '../components/icons';
import { RootStackParamList } from '../types';
import { ExploreService } from '../services/explore';
import { ReadService } from '../services/read';

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
  const [bookmarksCount, setBookmarksCount] = useState(0);
  const [upVotesCount, setUpVotesCount] = useState(0);
  const [downVotesCount, setDownVotesCount] = useState(0);

  const accountName = user?.email || 'Anonymous user';

  // Fetch user statistics
  useEffect(() => {
    const fetchStats = async () => {
      try {
        setLoading(true);
        setError(null);

        // Fetch all stats in parallel
        const [voteStats, subscriptions, bookmarks] = await Promise.all([
          ExploreService.getUserVoteStats(),
          ReadService.listAllSubscriptions(), // Use new aggregated endpoint
          ReadService.listUserContents({ limit: 1 }), // Just get count, not all items
        ]);

        setUpVotesCount(voteStats.upvotes);
        setDownVotesCount(voteStats.downvotes);
        setFeedsCount(subscriptions.total_count); // Now includes all subscription types
        setBookmarksCount(bookmarks.total_count);
      } catch (err) {
        console.error('Error fetching user stats:', err);
        setError(err instanceof Error ? err.message : 'Failed to load statistics');
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
  }, []);

  const handleAccountPress = () => {
    navigation.navigate('Account');
  };

  const handleFeedsPress = () => {
    // TODO: Navigate to feeds management
    console.log('Feeds pressed');
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
            subtitle={loading ? 'Loading...' : `${feedsCount} subscriptions`}
            onPress={handleFeedsPress}
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
                : `${upVotesCount} up votes, ${downVotesCount} down votes`
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
