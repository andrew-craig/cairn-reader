import React from 'react';
import { StyleSheet, useColorScheme, Platform } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { BlurView } from 'expo-blur';
import { LinearGradient } from 'expo-linear-gradient';
import MaskedView from '@react-native-masked-view/masked-view';

/**
 * TopBlurGradient - Gradual blur overlay for the status bar area
 *
 * Uses MaskedView to create a blur that gradually fades from opaque at the top
 * to transparent below, making content readable when it scrolls behind the status bar.
 */
export const TopBlurGradient: React.FC = () => {
  const colorScheme = useColorScheme();
  const insets = useSafeAreaInsets();

  // Extend gradient below status bar for smooth fade
  const gradientHeight = 20;
  const totalHeight = insets.top + gradientHeight;

  return (
    <MaskedView
      style={[styles.container, { height: totalHeight }]}
      maskElement={
        <LinearGradient
          colors={['black', 'black', 'transparent']}
          locations={[0, 0.5, 1]}
          style={{ flex: 1 }}
        />
      }
      pointerEvents="none"
    >
      <BlurView
        intensity={Platform.OS === 'ios' ? 10 : 20}
        tint={colorScheme === 'dark' ? 'dark' : 'light'}
        style={styles.blur}
      />
    </MaskedView>
  );
};

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    zIndex: 1000,
  },
  blur: {
    flex: 1,
  },
});
