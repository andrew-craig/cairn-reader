import React from 'react';
import { render, screen } from '@testing-library/react-native';
import RootNavigator from './RootNavigator';
import { useAuth } from '../contexts/AuthContext';
import { useNetworkStatus } from '../hooks/useNetworkStatus';
import { useSyncTrigger } from '../hooks/useSyncTrigger';

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
jest.mock('../hooks/useSyncTrigger');
jest.mock('./TabNavigator', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  return {
    TabNavigator: () => <Text>MainTabsScreen</Text>,
  };
});
// Only LoginScreen is stubbed — it's what actually renders in the
// unauthenticated branch below, unlike the other '../screens' exports,
// which the mocked Stack.Navigator above never invokes.
jest.mock('../screens', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const actual = jest.requireActual('../screens');
  return {
    ...actual,
    LoginScreen: () => <Text>LoginScreen</Text>,
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

  it('mounts the sync trigger when authenticated', () => {
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: false });
    render(<RootNavigator />);

    expect(useSyncTrigger).toHaveBeenCalled();
  });

  it('does not mount the sync trigger when unauthenticated', () => {
    (useAuth as jest.Mock).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
      login: jest.fn(),
    });
    render(<RootNavigator />);

    expect(screen.getByText('LoginScreen')).toBeTruthy();
    expect(useSyncTrigger).not.toHaveBeenCalled();
  });

  // task_5bd6: RootNavigator's isLoading and !isAuthenticated early returns
  // never rendered OfflineBanner, so a login attempt (or the loading spinner
  // itself) gave no connectivity indication. SyncTriggerEffect must stay out
  // of both — its doc comment is explicit that living inside the
  // authenticated tree only is what keeps the sync trigger from running
  // logged out.
  it('shows the offline banner on the loading branch, without mounting the sync trigger', () => {
    (useAuth as jest.Mock).mockReturnValue({
      isAuthenticated: false,
      isLoading: true,
      login: jest.fn(),
    });
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: true });
    render(<RootNavigator />);

    expect(screen.getByText(/offline/i)).toBeTruthy();
    expect(screen.queryByText('LoginScreen')).toBeNull();
    expect(screen.queryByText('MainTabsScreen')).toBeNull();
    expect(useSyncTrigger).not.toHaveBeenCalled();
  });

  it('shows the offline banner on the login branch, without mounting the sync trigger', () => {
    (useAuth as jest.Mock).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
      login: jest.fn(),
    });
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: true });
    render(<RootNavigator />);

    expect(screen.getByText(/offline/i)).toBeTruthy();
    expect(screen.getByText('LoginScreen')).toBeTruthy();
    expect(useSyncTrigger).not.toHaveBeenCalled();
  });

  // task_cab7 (see LEARNINGS.md, 2026-09-06) made a cold start offline with
  // valid persisted tokens keep the user signed in — AuthContext resolves
  // isAuthenticated: true despite isOffline: true. RootNavigator must route
  // that state to the authenticated tree, never to LoginScreen. AuthContext's
  // own resolution of that state is covered separately in
  // AuthContext.test.tsx; this asserts RootNavigator's side of the contract.
  it('cold start offline with valid tokens renders the authenticated tree, not LoginScreen', () => {
    (useAuth as jest.Mock).mockReturnValue({
      isAuthenticated: true,
      isLoading: false,
      login: jest.fn(),
    });
    (useNetworkStatus as jest.Mock).mockReturnValue({ isOffline: true });
    render(<RootNavigator />);

    expect(screen.getByText('MainTabsScreen')).toBeTruthy();
    expect(screen.queryByText('LoginScreen')).toBeNull();
  });
});
