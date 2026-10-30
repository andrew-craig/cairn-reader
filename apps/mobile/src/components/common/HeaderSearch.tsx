import React, { useEffect, useRef, useState } from 'react';
import {
  AccessibilityInfo,
  StyleProp,
  ViewStyle,
  Animated,
  Easing,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  useColorScheme,
  View,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Colors, FontFamily, FontSizes } from '../../constants';

export interface HeaderSearchProps {
  open: boolean;
  query: string;
  onChangeQuery: (query: string) => void;
  onOpen: () => void;
  onClose: () => void;
  placeholder: string;
}

const BUTTON = 42;
// The pill reaches this far past the buttons on every side so its ring always
// sits outside their pressed state.
const INSET = 3;
const PILL = BUTTON + INSET * 2;

const easeOut = Easing.bezier(0.23, 1, 0.32, 1);
const easeInOut = Easing.bezier(0.77, 0, 0.175, 1);

/**
 * Header search (mirrors the web list header): the round search button morphs
 * into a pill-shaped field. Opening first rings the circle, then stretches it
 * leftwards to `openWidth`; the search icon rides the pill's left edge into the
 * leading position and a close button appears where the icon started. Closing
 * plays the same steps in reverse, faster. Reduced motion snaps the size and
 * fades the contents.
 *
 * Renders a button-sized footprint; the pill is positioned over it and grows
 * leftwards to `leftEdge` (an x offset in the footprint's parent) over whatever
 * is beside it, so it has an opaque background.
 */
export const HeaderSearch: React.FC<
  HeaderSearchProps & { leftEdge: number; style?: StyleProp<ViewStyle> }
> = ({
  open,
  query,
  onChangeQuery,
  onOpen,
  onClose,
  placeholder,
  leftEdge,
  style,
}) => {
  const colors = Colors[useColorScheme() ?? 'light'];
  const inputRef = useRef<TextInput>(null);
  const [focused, setFocused] = useState(false);
  const [reduceMotion, setReduceMotion] = useState(false);
  const [footprintX, setFootprintX] = useState<number | null>(null);
  // Open, the pill also reaches INSET past `leftEdge`, so its rounded end
  // covers whatever starts there.
  const openWidth = footprintX === null ? PILL : footprintX + BUTTON + INSET * 2 - leftEdge;

  const width = useRef(new Animated.Value(open ? openWidth : PILL)).current;
  const ring = useRef(new Animated.Value(open ? 1 : 0)).current;
  const input = useRef(new Animated.Value(open ? 1 : 0)).current;
  const close = useRef(new Animated.Value(open ? 1 : 0)).current;

  useEffect(() => {
    AccessibilityInfo.isReduceMotionEnabled().then(setReduceMotion);
    const sub = AccessibilityInfo.addEventListener('reduceMotionChanged', setReduceMotion);
    return () => sub.remove();
  }, []);

  useEffect(() => {
    const timing = (
      value: Animated.Value,
      toValue: number,
      duration: number,
      delay = 0,
      easing = easeOut,
      useNativeDriver = true,
    ) => Animated.timing(value, { toValue, duration, delay, easing, useNativeDriver });

    let animation: Animated.CompositeAnimation;
    if (reduceMotion) {
      width.setValue(open ? openWidth : PILL);
      ring.setValue(open ? 1 : 0);
      animation = Animated.parallel([
        timing(input, open ? 1 : 0, 150),
        timing(close, open ? 1 : 0, 150),
      ]);
    } else if (open) {
      animation = Animated.parallel([
        timing(ring, 1, 120),
        timing(width, openWidth, 260, 80, easeInOut, false),
        timing(input, 1, 200, 240),
        timing(close, 1, 160, 280),
      ]);
    } else {
      animation = Animated.parallel([
        timing(input, 0, 100),
        timing(close, 0, 100),
        timing(width, PILL, 220, 40, easeOut, false),
        timing(ring, 0, 120, 200),
      ]);
    }
    animation.start();
    return () => animation.stop();
  }, [open, openWidth, reduceMotion, width, ring, input, close]);

  // Focus follows the toggle: the field on open, nothing on close.
  useEffect(() => {
    if (open) inputRef.current?.focus();
    else inputRef.current?.blur();
  }, [open]);

  return (
    <View style={[styles.footprint, style]} onLayout={(e) => setFootprintX(e.nativeEvent.layout.x)}>
      <Animated.View style={[styles.pill, { width, backgroundColor: colors.background }]}>
        <Animated.View
          pointerEvents="none"
          style={[
            styles.ring,
            { opacity: ring, borderColor: focused ? colors.primary : colors.textSecondary },
          ]}
        />
        <Animated.View
          style={[StyleSheet.absoluteFill, { opacity: input }]}
          pointerEvents={open ? 'auto' : 'none'}
          accessibilityElementsHidden={!open}
          importantForAccessibility={open ? 'auto' : 'no-hide-descendants'}
        >
          <TextInput
            ref={inputRef}
            style={[styles.input, { color: colors.text }]}
            placeholder={placeholder}
            placeholderTextColor={colors.textSecondary}
            accessibilityLabel={placeholder}
            value={query}
            onChangeText={onChangeQuery}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            autoCapitalize="none"
            autoCorrect={false}
            returnKeyType="search"
          />
        </Animated.View>
        <TouchableOpacity
          style={[styles.button, styles.toggle]}
          onPress={onOpen}
          disabled={open}
          activeOpacity={0.7}
          accessibilityRole="button"
          accessibilityLabel="Search"
          accessibilityState={{ expanded: open }}
          accessibilityElementsHidden={open}
          importantForAccessibility={open ? 'no-hide-descendants' : 'auto'}
        >
          <Ionicons name="search-outline" size={24} color={open ? colors.textSecondary : colors.text} />
        </TouchableOpacity>
        <Animated.View
          style={[
            styles.button,
            styles.close,
            {
              opacity: close,
              transform: [{ scale: close.interpolate({ inputRange: [0, 1], outputRange: [0.6, 1] }) }],
            },
          ]}
          pointerEvents={open ? 'auto' : 'none'}
          accessibilityElementsHidden={!open}
          importantForAccessibility={open ? 'auto' : 'no-hide-descendants'}
        >
          <TouchableOpacity
            style={styles.closeTouch}
            onPress={onClose}
            activeOpacity={0.7}
            accessibilityRole="button"
            accessibilityLabel="Close search"
          >
            <Ionicons name="close-outline" size={24} color={colors.text} />
          </TouchableOpacity>
        </Animated.View>
      </Animated.View>
    </View>
  );
};

const styles = StyleSheet.create({
  footprint: {
    width: BUTTON,
    height: BUTTON,
  },
  pill: {
    position: 'absolute',
    top: -INSET,
    right: -INSET,
    height: PILL,
    borderRadius: PILL / 2,
    overflow: 'hidden',
  },
  ring: {
    ...StyleSheet.absoluteFillObject,
    borderWidth: 1,
    borderRadius: PILL / 2,
  },
  input: {
    flex: 1,
    paddingHorizontal: BUTTON + INSET,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
  },
  button: {
    position: 'absolute',
    top: INSET,
    width: BUTTON,
    height: BUTTON,
    borderRadius: BUTTON / 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
  toggle: {
    left: INSET,
  },
  close: {
    right: INSET,
  },
  closeTouch: {
    width: BUTTON,
    height: BUTTON,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
