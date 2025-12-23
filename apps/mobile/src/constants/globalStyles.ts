import { StyleSheet } from 'react-native';
import { Spacing, FontSizes, BorderRadius, FontFamily } from './theme';

export const GlobalStyles = StyleSheet.create({
  // Container styles
  container: {
    flex: 1,
  },
  centerContent: {
    alignItems: 'center',
    justifyContent: 'center',
  },

  // Text styles
  text: {
    fontFamily: FontFamily.default,
    fontSize: FontSizes.md,
  },
  textSecondary: {
    fontFamily: FontFamily.default,
    fontSize: FontSizes.sm,
  },
  textBold: {
    fontFamily: FontFamily.defaultBold,
  },
  textSemiBold: {
    fontFamily: FontFamily.defaultSemiBold,
  },
  textMedium: {
    fontFamily: FontFamily.defaultMedium,
  },

  // Heading styles
  heading: {
    fontFamily: FontFamily.heading,
  },
  headingBold: {
    fontFamily: FontFamily.headingBold,
  },
  headingSemiBold: {
    fontFamily: FontFamily.headingSemiBold,
  },

  // Section styles
  section: {
    marginTop: Spacing.lg,
    marginHorizontal: Spacing.md,
  },
  sectionTitle: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.defaultSemiBold,
    textTransform: 'uppercase',
    marginBottom: Spacing.sm,
    marginLeft: Spacing.sm,
  },

  // Card styles
  card: {
    borderRadius: BorderRadius.lg,
    overflow: 'hidden',
  },

  // Row styles
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: Spacing.md,
  },
  rowLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.md,
  },

  // Input styles
  input: {
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.md,
    borderRadius: BorderRadius.md,
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    borderWidth: 1,
    minHeight: 48,
  },

  // Header styles
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.md,
    height: 64,
  },
  headerLeft: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
  },
  headerTitle: {
    fontSize: FontSizes.xl,
    fontFamily: FontFamily.defaultSemiBold,
  },
  headerRight: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: Spacing.sm,
  },

  // Empty state styles
  emptyContainer: {
    padding: Spacing.xxl,
    alignItems: 'center',
    justifyContent: 'center',
  },
  emptyText: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    textAlign: 'center',
  },

  // Footer styles
  footer: {
    alignItems: 'center',
    marginTop: Spacing.xxl,
    marginBottom: Spacing.xl,
  },
  footerText: {
    fontSize: FontSizes.sm,
    fontFamily: FontFamily.default,
    marginBottom: Spacing.xs,
  },

  // Button container
  buttonContainer: {
    gap: Spacing.md,
    marginVertical: Spacing.lg,
    width: '100%',
    maxWidth: 400,
  },
});
