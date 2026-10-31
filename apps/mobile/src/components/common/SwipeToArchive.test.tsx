import React from 'react';
import { Text } from 'react-native';
import { act, fireEvent, render, screen } from '@testing-library/react-native';
import { State } from 'react-native-gesture-handler';
import { fireGestureHandler } from 'react-native-gesture-handler/jest-utils';
import { SwipeToArchive, ARCHIVE_THRESHOLD } from './SwipeToArchive';

const ROW_WIDTH = 400;

// Swipeable sizes its action panel from layout events, which jest never emits,
// so the test fires them by hand. Which nodes carry `onLayout` is internal to
// react-native-gesture-handler's Swipeable (written against 2.28): the first is
// the row container, the last the action panel's left-edge marker. The length
// guard makes a gesture-handler upgrade that reshapes that tree fail here,
// rather than as a misleading "did not archive".
const renderRow = (onArchive: () => void) => {
  const utils = render(
    <SwipeToArchive onArchive={onArchive}>
      <Text>Row</Text>
    </SwipeToArchive>,
  );
  const layoutNodes = screen.UNSAFE_root.findAll(
    (node: { props: Record<string, unknown> }) => typeof node.props.onLayout === 'function',
  );
  expect(layoutNodes).toHaveLength(5);
  fireEvent(layoutNodes[0], 'layout', { nativeEvent: { layout: { x: 0, y: 0, width: ROW_WIDTH, height: 80 } } });
  fireEvent(layoutNodes[layoutNodes.length - 1], 'layout', { nativeEvent: { layout: { x: 0, y: 0, width: 0, height: 0 } } });
  return utils;
};

const swipe = (translationX: number) => {
  fireGestureHandler(screen.getByTestId('swipe-to-archive'), [
    { state: State.BEGAN, translationX: 0, velocityX: 0 },
    { state: State.ACTIVE, translationX: translationX / 2, velocityX: 0 },
    { state: State.ACTIVE, translationX, velocityX: 0 },
    { state: State.END, translationX, velocityX: 0 },
  ]);
};

describe('SwipeToArchive', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('archives when released past the full width of the icon', () => {
    const onArchive = jest.fn();
    renderRow(onArchive);

    swipe(-(ARCHIVE_THRESHOLD + 10));
    act(() => jest.runAllTimers());

    expect(onArchive).toHaveBeenCalledTimes(1);
  });

  it('springs back without archiving when released short of the icon', () => {
    const onArchive = jest.fn();
    renderRow(onArchive);

    swipe(-(ARCHIVE_THRESHOLD - 10));
    act(() => jest.runAllTimers());

    expect(onArchive).not.toHaveBeenCalled();
  });
});
