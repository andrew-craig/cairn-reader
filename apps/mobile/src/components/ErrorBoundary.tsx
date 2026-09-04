import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet, useColorScheme } from 'react-native';
import { Colors, Spacing, FontSizes, FontFamily, BorderRadius } from '../constants';

interface Props {
  children: React.ReactNode;
}

interface State {
  hasError: boolean;
}

function ErrorFallback({ onRetry }: { onRetry: () => void }): React.ReactElement {
  const colors = Colors[useColorScheme() ?? 'light'];

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <Text style={[styles.title, { color: colors.text }]}>Something went wrong</Text>
      <Text style={[styles.text, { color: colors.textSecondary }]}>
        The app hit an unexpected error and needs to restart.
      </Text>
      <TouchableOpacity
        style={[styles.button, { backgroundColor: colors.primary }]}
        onPress={onRetry}
        accessibilityRole="button"
        accessibilityLabel="Restart app"
      >
        <Text style={[styles.buttonText, { color: colors.background }]}>Restart app</Text>
      </TouchableOpacity>
    </View>
  );
}

/**
 * Catches render/lifecycle errors anywhere below it so a single malformed
 * article (or any unexpected throw) shows a recoverable message instead of
 * crashing the app to a blank screen. "Restart app" clears the error and
 * re-renders the subtree from scratch (the boundary wraps AuthProvider and
 * NavigationContainer, so this resets navigation and auth state too).
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

    return <ErrorFallback onRetry={this.handleRetry} />;
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
