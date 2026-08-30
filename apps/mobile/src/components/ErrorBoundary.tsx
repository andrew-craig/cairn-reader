import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet, Appearance } from 'react-native';
import { Colors, Spacing, FontSizes, FontFamily, BorderRadius } from '../constants';

interface Props {
  children: React.ReactNode;
}

interface State {
  hasError: boolean;
}

/**
 * Catches render/lifecycle errors anywhere below it so a single malformed
 * article (or any unexpected throw) shows a recoverable message instead of
 * crashing the app to a blank screen. "Try again" clears the error and
 * re-renders the subtree.
 */
export class ErrorBoundary extends React.Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    console.error('Unhandled render error:', error, info.componentStack);
  }

  private handleRetry = () => {
    this.setState({ hasError: false });
  };

  render(): React.ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }

    const colors = Colors[Appearance.getColorScheme() ?? 'light'];

    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <Text style={[styles.title, { color: colors.text }]}>Something went wrong</Text>
        <Text style={[styles.text, { color: colors.textSecondary }]}>
          The app hit an unexpected error.
        </Text>
        <TouchableOpacity
          style={[styles.button, { backgroundColor: colors.primary }]}
          onPress={this.handleRetry}
          accessibilityRole="button"
          accessibilityLabel="Try again"
        >
          <Text style={[styles.buttonText, { color: colors.background }]}>Try again</Text>
        </TouchableOpacity>
      </View>
    );
  }
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: Spacing.xl,
    gap: Spacing.sm,
  },
  title: {
    fontFamily: FontFamily.headingBold,
    fontSize: FontSizes.xl,
    textAlign: 'center',
  },
  text: {
    fontFamily: FontFamily.default,
    fontSize: FontSizes.sm,
    textAlign: 'center',
  },
  button: {
    marginTop: Spacing.md,
    paddingVertical: Spacing.sm,
    paddingHorizontal: Spacing.lg,
    borderRadius: BorderRadius.md,
  },
  buttonText: {
    fontFamily: FontFamily.defaultSemiBold,
    fontSize: FontSizes.sm,
  },
});
