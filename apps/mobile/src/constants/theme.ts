// Generated from design/tokens/colors.json (shared with the web app).
export { Colors } from './colors';

export const Spacing = {
  xs: 4,
  sm: 8,
  md: 16,
  lg: 24,
  xl: 32,
  xxl: 48,
};

export const FontSizes = {
  xs: 12,
  sm: 14,
  md: 16,
  lg: 18,
  xl: 24,
  xxl: 32,
};

export const BorderRadius = {
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
  full: 9999,
};

export const FontFamily = {
  default: 'Inter_400Regular',
  defaultMedium: 'Inter_500Medium',
  defaultSemiBold: 'Inter_600SemiBold',
  defaultBold: 'Inter_700Bold',
  heading: 'CrimsonPro_400Regular',
  headingMedium: 'CrimsonPro_500Medium',
  headingSemiBold: 'CrimsonPro_600SemiBold',
  headingBold: 'CrimsonPro_700Bold',
};

/**
 * Safe Area Layout Constants
 *
 * These values define the fixed heights of floating UI elements (tab bar, action menu)
 * that sit above the bottom safe area. Use these with useSafeAreaInsets() to calculate
 * proper content padding.
 *
 * Example usage:
 *   const insets = useSafeAreaInsets();
 *   const bottomPadding = Layout.tabBarHeight + insets.bottom + Spacing.md;
 */
export const Layout = {
  /** Height of the tab bar pill (54px) + vertical padding (8px top + 8px bottom) */
  tabBarHeight: 70,
  /** Height of the bottom action menu pill (54px) + vertical padding (8px top + 8px bottom) */
  bottomActionMenuHeight: 70,
  /** Standard header height including vertical padding */
  headerHeight: 64,
};
