import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  useColorScheme,
  TouchableOpacity,
  Alert,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import { StorageService } from '../services';
import { Colors, Spacing, FontSizes, BorderRadius, GlobalStyles, FontFamily, Layout } from '../constants';

export const SettingsScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();

  // Calculate padding to account for tab bar + bottom safe area
  const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;

  const handleClearAllData = () => {
    Alert.alert(
      'Clear All Data',
      'Are you sure you want to delete all articles? This cannot be undone.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete All',
          style: 'destructive',
          onPress: async () => {
            try {
              await StorageService.clearAllArticles();
              Alert.alert('Success', 'All articles have been deleted');
            } catch (error) {
              console.error('Failed to clear data:', error);
              Alert.alert('Error', 'Failed to delete articles');
            }
          },
        },
      ]
    );
  };

  return (
    <ScrollView
      style={[GlobalStyles.container, { backgroundColor: colors.background }]}
      contentContainerStyle={{
        paddingTop: insets.top + Spacing.md,
        paddingBottom: bottomPadding,
      }}
    >
      <View style={GlobalStyles.section}>
        <Text style={[GlobalStyles.sectionTitle, { color: colors.text }]}>
          About
        </Text>
        <View style={[GlobalStyles.card, { backgroundColor: colors.card }]}>
          <View style={GlobalStyles.row}>
            <Text style={[styles.label, { color: colors.text }]}>Version</Text>
            <Text style={[styles.value, { color: colors.textSecondary }]}>
              1.0.0
            </Text>
          </View>
        </View>
      </View>

      <View style={GlobalStyles.section}>
        <Text style={[GlobalStyles.sectionTitle, { color: colors.text }]}>
          Data
        </Text>
        <View style={[GlobalStyles.card, { backgroundColor: colors.card }]}>
          <TouchableOpacity
            style={GlobalStyles.row}
            onPress={handleClearAllData}
          >
            <View style={GlobalStyles.rowLeft}>
              <Ionicons name="trash-outline" size={24} color={colors.error} />
              <Text style={[styles.label, styles.dangerText, { color: colors.error }]}>
                Clear All Articles
              </Text>
            </View>
            <Ionicons
              name="chevron-forward"
              size={24}
              color={colors.textSecondary}
            />
          </TouchableOpacity>
        </View>
      </View>

      <View style={GlobalStyles.footer}>
        <Text style={[GlobalStyles.footerText, { color: colors.textSecondary }]}>
          ReadItLater
        </Text>
        <Text style={[GlobalStyles.footerText, { color: colors.textSecondary }]}>
          Save articles for later reading
        </Text>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  label: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  value: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  dangerText: {
    marginLeft: Spacing.sm,
  },
});
