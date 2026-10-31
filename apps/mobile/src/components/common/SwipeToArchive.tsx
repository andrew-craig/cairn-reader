import React, { ReactNode } from 'react';
import { Animated, StyleSheet, View, useColorScheme } from 'react-native';
import Swipeable from 'react-native-gesture-handler/Swipeable';
import { Colors, Spacing } from '../../constants';
import { ArchiveIcon } from '../icons';

const ICON_SIZE = 24;
// Distance the row must travel to fully uncover the icon. Releasing past it
// archives; releasing short of it springs the row back.
export const ARCHIVE_THRESHOLD = Spacing.md + ICON_SIZE;

interface SwipeToArchiveProps {
  children: ReactNode;
  onArchive: () => void;
}

/**
 * Wraps a list row so swiping it right-to-left archives it. The archive action
 * spans the full row, so a confirmed swipe carries the row off-screen before
 * `onArchive` fires.
 */
export const SwipeToArchive: React.FC<SwipeToArchiveProps> = ({ children, onArchive }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const renderRightActions = (
    _progress: Animated.AnimatedInterpolation<number>,
    dragX: Animated.AnimatedInterpolation<number>,
  ) => {
    // Step from the outline icon to the solid one once the release would archive.
    const solidOpacity = dragX.interpolate({
      inputRange: [-ARCHIVE_THRESHOLD - 1, -ARCHIVE_THRESHOLD],
      outputRange: [1, 0],
      extrapolate: 'clamp',
    });
    const outlineOpacity = dragX.interpolate({
      inputRange: [-ARCHIVE_THRESHOLD - 1, -ARCHIVE_THRESHOLD],
      outputRange: [0, 1],
      extrapolate: 'clamp',
    });

    return (
      <View style={[styles.action, { backgroundColor: colors.success }]}>
        <View style={styles.icon}>
          <Animated.View style={{ opacity: outlineOpacity }}>
            <ArchiveIcon size={ICON_SIZE} color="#FFFFFF" />
          </Animated.View>
          <Animated.View style={[StyleSheet.absoluteFill, { opacity: solidOpacity }]}>
            <ArchiveIcon size={ICON_SIZE} color="#FFFFFF" filled />
          </Animated.View>
        </View>
      </View>
    );
  };

  return (
    <Swipeable
      renderRightActions={renderRightActions}
      rightThreshold={ARCHIVE_THRESHOLD}
      overshootRight={false}
      friction={1}
      onSwipeableOpen={onArchive}
      testID="swipe-to-archive"
      childrenContainerStyle={{ backgroundColor: colors.background }}
    >
      {children}
    </Swipeable>
  );
};

const styles = StyleSheet.create({
  action: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'flex-end',
    paddingRight: Spacing.md,
  },
  icon: {
    width: ICON_SIZE,
    height: ICON_SIZE,
  },
});
