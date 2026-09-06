import React from 'react';
import { View, Text, StyleSheet, useColorScheme } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Colors, Spacing, FontSizes, FontFamily } from '../../constants/theme';
import { useNetworkStatus } from '../../hooks/useNetworkStatus';

/**
 * Absolutely-positioned overlay, not a layout row — see the Safe Area
 * Strategy in apps/mobile/CLAUDE.md. Every screen scrolls content behind the
 * status bar and pads its own header by insets.top; a banner in the normal
 * layout flow above the navigator would reflow every screen. Rendered as a
 * sibling after Stack.Navigator so it paints on top without reflowing
 * anything. It covers the header area while offline — accepted tradeoff.
 */
export const OfflineBanner: React.FC = () => {
  const { isOffline } = useNetworkStatus();
  const insets = useSafeAreaInsets();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  if (!isOffline) {
    return null;
  }

  return (
    <View
      style={[styles.container, { paddingTop: insets.top, backgroundColor: colors.warning }]}
      pointerEvents="none"
    >
      <Text style={[styles.text, { color: colors.background }]}>You&rsquo;re offline</Text>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    // Above TopBlurGradient's overlay (zIndex 1000) so the banner text isn't
    // obscured by the status-bar blur.
    zIndex: 1001,
    alignItems: 'center',
    justifyContent: 'center',
    paddingBottom: Spacing.xs,
  },
  text: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.defaultMedium,
  },
});
