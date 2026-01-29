import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity, ScrollView, useColorScheme } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuth } from '../contexts/AuthContext';
import { Colors, Spacing, FontSizes, FontFamily, Layout } from '../constants/theme';
import { GlobalStyles } from '../constants/globalStyles';
import { ChevronRightIcon } from '../components/icons';

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
      style={[styles.menuItem, { backgroundColor: colors.background }]}
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
      <ChevronRightIcon size={20} color={colors.text} />
    </TouchableOpacity>
  );
};

export const YouScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const { user, logout } = useAuth();
  const colors = isDark ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  // Calculate bottom padding to account for tab bar + bottom safe area
  const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;

  // Placeholder data - these will be replaced with actual data from API
  const accountName = user?.email || 'Anonymous user';
  const feedsCount = 8;
  const bookmarksCount = 98;
  const upVotesCount = 8;
  const downVotesCount = 5;

  const handleAccountPress = () => {
    // TODO: Navigate to account settings
    console.log('Account pressed');
  };

  const handleFeedsPress = () => {
    // TODO: Navigate to feeds management
    console.log('Feeds pressed');
  };

  const handleBookmarksPress = () => {
    // TODO: Navigate to bookmarks
    console.log('Bookmarks pressed');
  };

  const handleVotesPress = () => {
    // TODO: Navigate to votes history
    console.log('Votes pressed');
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
            },
          ]}
        >
          <Text style={[GlobalStyles.headerTitle, { color: colors.text }]}>You</Text>
        </View>

        <View style={styles.menuSection}>
          <MenuItem
            title="Account"
            subtitle={accountName}
            onPress={handleAccountPress}
            isDark={isDark}
          />
          <MenuItem
            title="Feeds"
            subtitle={`${feedsCount} subscriptions`}
            onPress={handleFeedsPress}
            isDark={isDark}
          />
          <MenuItem
            title="Bookmarks"
            subtitle={`${bookmarksCount} saved`}
            onPress={handleBookmarksPress}
            isDark={isDark}
          />
          <MenuItem
            title="Votes"
            subtitle={`${upVotesCount} up votes, ${downVotesCount} down votes`}
            onPress={handleVotesPress}
            isDark={isDark}
          />
        </View>

        <View style={styles.logoutSection}>
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
    height: 64,
    justifyContent: 'center',
  },
  menuSection: {
    marginTop: Spacing.lg,
  },
  logoutSection: {
    marginTop: Spacing.xl,
  },
  menuItem: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    minHeight: 60,
  },
  menuItemContent: {
    flex: 1,
    marginRight: Spacing.md,
  },
  menuItemTitle: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    marginBottom: 2,
  },
  menuItemSubtitle: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    marginTop: 2,
  },
});
