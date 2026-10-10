import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { TabNavigator } from './TabNavigator';

// The tabs are Feed | Reads | You (Explore is gone). The bar and screens are
// stubbed: this pins the tab set and order, not their content.

jest.mock('../screens', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const ReactModule = require('react');
  const stub = (name: string) => () => ReactModule.createElement(Text, null, `screen:${name}`);
  return {
    FeedScreen: stub('Feed'),
    ReadsScreen: stub('Reads'),
    YouScreen: stub('You'),
  };
});

// Same safe-area-context mock gap as RootNavigator.test.tsx keeps the real
// bottom-tabs navigator from rendering here, so it is stubbed to list the
// declared screens and render the first (the initial tab).
jest.mock('@react-navigation/bottom-tabs', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const ReactModule = require('react');
  return {
    createBottomTabNavigator: () => ({
      Navigator: ({ children }: { children: React.ReactNode }) => {
        const screens = ReactModule.Children.toArray(children) as React.ReactElement<{
          name: string;
          component: React.ComponentType;
        }>[];
        const Initial = screens[0].props.component;
        return ReactModule.createElement(
          ReactModule.Fragment,
          null,
          ReactModule.createElement(Text, null, `tabs:${screens.map((s) => s.props.name).join(',')}`),
          ReactModule.createElement(Initial),
        );
      },
      Screen: () => null,
    }),
  };
});

describe('TabNavigator', () => {
  it('has Feed, Reads and You tabs, in that order', () => {
    render(<TabNavigator />);

    expect(screen.getByText('tabs:Feed,Reads,You')).toBeTruthy();
  });

  it('opens on the Feed tab', () => {
    render(<TabNavigator />);

    expect(screen.getByText('screen:Feed')).toBeTruthy();
  });
});
