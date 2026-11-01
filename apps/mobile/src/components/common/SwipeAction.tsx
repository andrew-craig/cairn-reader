import React, { ReactNode } from 'react';
import { Animated, StyleSheet, View, useColorScheme } from 'react-native';
import Swipeable from 'react-native-gesture-handler/Swipeable';
import { Colors, Spacing } from '../../constants';

const ICON_SIZE = 24;
// Distance the row must travel to fully uncover the icon. Releasing past it
// triggers the action; releasing short of it springs the row back.
export const SWIPE_ACTION_THRESHOLD = Spacing.md + ICON_SIZE;

interface SwipeActionProps {
  children: ReactNode;
  /** Direction of the swipe: 'left' is right-to-left, 'right' is left-to-right. */
  direction: 'left' | 'right';
  /** Icon revealed under the row; switches to its `filled` variant once a release would trigger. */
  icon: React.ComponentType<{ size?: number; color?: string; filled?: boolean }>;
  /** Theme colour revealed under the row. */
  color: keyof typeof Colors.light;
  /**
   * Fires once the row has been carried off-screen. If it returns a promise
   * that rejects, the row springs back (the caller reports the error).
   */
  onAction: () => void | Promise<void>;
}

/**
 * Wraps a list row so swiping it in `direction` triggers `onAction`. The action
 * spans the full row, so a confirmed swipe carries the row off-screen before
 * `onAction` fires.
 */
export const SwipeAction: React.FC<SwipeActionProps> = ({
  children,
  direction,
  icon: Icon,
  color,
  onAction,
}) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  // Swiping left drags the row to negative x and uncovers Swipeable's right
  // actions; swiping right is the mirror image.
  const sign = direction === 'left' ? -1 : 1;

  const renderActions = (
    _progress: Animated.AnimatedInterpolation<number>,
    dragX: Animated.AnimatedInterpolation<number>,
  ) => {
    // Step from the outline icon to the solid one once the release would trigger.
    const distance = Animated.multiply(dragX, sign);
    const inputRange = [SWIPE_ACTION_THRESHOLD, SWIPE_ACTION_THRESHOLD + 1];
    const solidOpacity = distance.interpolate({ inputRange, outputRange: [0, 1], extrapolate: 'clamp' });
    const outlineOpacity = distance.interpolate({ inputRange, outputRange: [1, 0], extrapolate: 'clamp' });

    return (
      <View
        style={[
          styles.action,
          direction === 'left' ? styles.actionLeft : styles.actionRight,
          { backgroundColor: colors[color] },
        ]}
      >
        <View style={styles.icon}>
          <Animated.View style={{ opacity: outlineOpacity }}>
            <Icon size={ICON_SIZE} color="#FFFFFF" />
          </Animated.View>
          <Animated.View style={[StyleSheet.absoluteFill, { opacity: solidOpacity }]}>
            <Icon size={ICON_SIZE} color="#FFFFFF" filled />
          </Animated.View>
        </View>
      </View>
    );
  };

  const handleOpen = (_side: 'left' | 'right', swipeable: Swipeable) => {
    Promise.resolve(onAction()).catch(() => swipeable.close());
  };

  return (
    <Swipeable
      {...(direction === 'left'
        ? { renderRightActions: renderActions, rightThreshold: SWIPE_ACTION_THRESHOLD, overshootRight: false }
        : { renderLeftActions: renderActions, leftThreshold: SWIPE_ACTION_THRESHOLD, overshootLeft: false })}
      friction={1}
      onSwipeableOpen={handleOpen}
      testID="swipe-action"
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
  },
  actionLeft: {
    alignItems: 'flex-end',
    paddingRight: Spacing.md,
  },
  actionRight: {
    alignItems: 'flex-start',
    paddingLeft: Spacing.md,
  },
  icon: {
    width: ICON_SIZE,
    height: ICON_SIZE,
  },
});
