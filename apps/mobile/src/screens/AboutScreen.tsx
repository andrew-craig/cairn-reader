import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, ScrollView, useColorScheme } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ScreenHeader } from '../components/common/ScreenHeader';
import { SystemService } from '../services/system';
import { Colors, Spacing, FontSizes, FontFamily } from '../constants/theme';
import { RootStackParamList } from '../types';
import appConfig from '../../app.json';

type AboutScreenNavigationProp = StackNavigationProp<RootStackParamList, 'About'>;

// The app's own version, from the Expo manifest (source of truth for the
// shipped build).
const appVersion = appConfig.expo.version;

export const AboutScreen: React.FC = () => {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === 'dark';
  const colors = isDark ? Colors.dark : Colors.light;
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<AboutScreenNavigationProp>();

  const [version, setVersion] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    SystemService.getServerVersion()
      .then((v) => {
        if (active) setVersion(v);
      })
      .catch((err) => {
        console.error('Error fetching server version:', err);
        if (active) setError(true);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  // The cloud backend may not report a version; degrade gracefully rather than
  // showing an empty value.
  const versionValue = loading
    ? 'Loading…'
    : error
      ? 'Unavailable'
      : version ?? 'Unknown';

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={[styles.scrollContent, { paddingBottom: insets.bottom + Spacing.lg }]}
      >
        <ScreenHeader title="About" onBack={() => navigation.goBack()} />

        <View style={styles.content}>
          <View style={[styles.row, styles.rowFirst, { borderColor: colors.border }]}>
            <Text style={[styles.label, { color: colors.text }]}>App version</Text>
            <Text style={[styles.value, { color: colors.textSecondary }]}>{appVersion}</Text>
          </View>
          <View style={[styles.row, { borderColor: colors.border }]}>
            <Text style={[styles.label, { color: colors.text }]}>Server version</Text>
            <Text style={[styles.value, { color: colors.textSecondary }]}>{versionValue}</Text>
          </View>
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
  content: {
    paddingHorizontal: Spacing.md,
    paddingTop: Spacing.lg,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: Spacing.md,
    borderBottomWidth: 1,
  },
  rowFirst: {
    borderTopWidth: 1,
  },
  label: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  value: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
});
