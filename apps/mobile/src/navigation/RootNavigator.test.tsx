import React from 'react';
import { render, screen } from '@testing-library/react-native';
import RootNavigator from './RootNavigator';
import { useAuth } from '../contexts/AuthContext';
import { useNetworkStatus } from '../hooks/useNetworkStatus';

// The offline banner is an overlay sibling of the Stack.Navigator (see
// apps/mobile/CLAUDE.md Safe Area Strategy) — it must appear/disappear
// without affecting whether the navigator's screens render.
//
// @react-navigation/stack's real Navigator can't render under this test
// environment's safe-area-context mock (it reaches into
// SafeAreaInsetsContext, which the shared jest mock doesn't provide), which
// is a pre-existing test-infra gap unrelated to this change. It's mocked
// here to a minimal Navigator that renders its first Screen's component, so
// the test still proves the banner is a sibling of the navigator's content
// rather than replacing it.
jest.mock('@react-navigation/stack', () => {
  // require() is necessary here: jest.mock() factories are hoisted above
  // this file's imports, so they cannot reference top-level import bindings.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const RN = require('react');
  return {
    createStackNavigator: () => ({
      Navigator: ({ children }: { children: React.ReactNode }) => {
        const [first] = RN.Children.toArray(children) as React.ReactElement<{ component: React.ComponentType }>[];
        const Component = first.props.component;
        return <Component />;
      },
      Screen: () => null,
    }),
    CardStyleInterpolators: { forVerticalIOS: jest.fn() },
  };
});
jest.mock('../contexts/AuthContext');
jest.mock('../hooks/useNetworkStatus');
jest.mock('./TabNavigator', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  return {
    TabNavigator: () => <Text>MainTabsScreen</Text>,
  };
});

describe('RootNavigator offline banner', () => {
  beforeEach(() => {
    (useAuth as jest.Mock).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      login: jest.fn(),
    });
  });

  afterEach(() => jest.clearAllMocks());

  it('shows the offline banner and still renders screens when offline', () => {
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: true });
    render(<RootNavigator />);

    expect(screen.getByText(/offline/i)).toBeTruthy();
    expect(screen.getByText('MainTabsScreen')).toBeTruthy();
  });

  it('hides the offline banner when online, screens still render', () => {
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: false });
    render(<RootNavigator />);

    expect(screen.queryByText(/offline/i)).toBeNull();
    expect(screen.getByText('MainTabsScreen')).toBeTruthy();
  });
});
