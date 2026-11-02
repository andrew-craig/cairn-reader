import React from 'react';
import { Text } from 'react-native';
import { act, fireEvent, render, screen } from '@testing-library/react-native';
import { State } from 'react-native-gesture-handler';
import { fireGestureHandler } from 'react-native-gesture-handler/jest-utils';
import { SwipeAction, SWIPE_ACTION_THRESHOLD } from './SwipeAction';
import { ArchiveIcon } from '../icons';

const ROW_WIDTH = 400;

// Swipeable sizes its action panel from layout events, which jest never emits,
// so the test fires them by hand. Which nodes carry `onLayout` is internal to
// react-native-gesture-handler's Swipeable (written against 2.28): the first is
// the row container, the last the action panel's edge marker (its `x` is the
// panel's inner edge: 0 for right actions, the panel width for left actions).
// The length guard makes a gesture-handler upgrade that reshapes that tree fail
// here, rather than as a misleading "did not trigger".
const renderRow = (direction: 'left' | 'right', onAction: () => void | Promise<void>) => {
  const utils = render(
    <SwipeAction direction={direction} icon={ArchiveIcon} color="success" onAction={onAction}>
      <Text>Row</Text>
    </SwipeAction>,
  );
  const layoutNodes = screen.UNSAFE_root.findAll(
    (node: { props: Record<string, unknown> }) => typeof node.props.onLayout === 'function',
  );
  expect(layoutNodes).toHaveLength(5);
  const markerX = direction === 'left' ? 0 : ROW_WIDTH;
  fireEvent(layoutNodes[0], 'layout', { nativeEvent: { layout: { x: 0, y: 0, width: ROW_WIDTH, height: 80 } } });
  fireEvent(layoutNodes[layoutNodes.length - 1], 'layout', { nativeEvent: { layout: { x: markerX, y: 0, width: 0, height: 0 } } });
  return utils;
};

const swipe = (translationX: number) => {
  fireGestureHandler(screen.getByTestId('swipe-action'), [
    { state: State.BEGAN, translationX: 0, velocityX: 0 },
    { state: State.ACTIVE, translationX: translationX / 2, velocityX: 0 },
    { state: State.ACTIVE, translationX, velocityX: 0 },
    { state: State.END, translationX, velocityX: 0 },
  ]);
  act(() => jest.runAllTimers());
};

describe('SwipeAction', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  describe.each([
    ['left', -1],
    ['right', 1],
  ] as const)('direction=%s', (direction, sign) => {
    it('triggers when released past the full width of the icon', () => {
      const onAction = jest.fn();
      renderRow(direction, onAction);

      swipe(sign * (SWIPE_ACTION_THRESHOLD + 10));

      expect(onAction).toHaveBeenCalledTimes(1);
    });

    it('springs back without triggering when released short of the icon', () => {
      const onAction = jest.fn();
      renderRow(direction, onAction);

      swipe(sign * (SWIPE_ACTION_THRESHOLD - 10));

      expect(onAction).not.toHaveBeenCalled();
    });

    it('ignores a swipe the other way', () => {
      const onAction = jest.fn();
      renderRow(direction, onAction);

      swipe(-sign * (SWIPE_ACTION_THRESHOLD + 10));

      expect(onAction).not.toHaveBeenCalled();
    });
  });

  // An open row re-fires on any further release, even a short one; a closed row
  // only on a full swipe. So a short follow-up swipe shows whether it closed.
  it('springs back when the action rejects', async () => {
    const onAction = jest.fn().mockRejectedValue(new Error('offline'));
    renderRow('right', onAction);

    swipe(SWIPE_ACTION_THRESHOLD + 10);
    await act(async () => {});
    swipe(10);

    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it('ignores further releases while the action is still pending', async () => {
    let resolve!: () => void;
    const onAction = jest.fn(() => new Promise<void>((r) => (resolve = r)));
    renderRow('right', onAction);

    swipe(SWIPE_ACTION_THRESHOLD + 10);
    swipe(10);
    expect(onAction).toHaveBeenCalledTimes(1);

    await act(async () => resolve());
    swipe(10);
    expect(onAction).toHaveBeenCalledTimes(2);
  });

  it('stays open when the action resolves', async () => {
    const onAction = jest.fn().mockResolvedValue(undefined);
    renderRow('right', onAction);

    swipe(SWIPE_ACTION_THRESHOLD + 10);
    await act(async () => {});
    swipe(10);

    expect(onAction).toHaveBeenCalledTimes(2);
  });
});
