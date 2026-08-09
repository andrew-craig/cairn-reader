import { throttle } from '@cairn/shared';

describe('throttle', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('calls the function immediately on the leading edge', () => {
    const fn = jest.fn();
    const throttled = throttle(fn, 1000);

    throttled('a');

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith('a');
  });

  it('coalesces calls within the interval into a single trailing call with the latest args', () => {
    const fn = jest.fn();
    const throttled = throttle(fn, 1000);

    throttled('a');
    throttled('b');
    throttled('c');

    expect(fn).toHaveBeenCalledTimes(1);

    jest.advanceTimersByTime(1000);

    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith('c');
  });

  it('fires again immediately once the interval has fully elapsed', () => {
    const fn = jest.fn();
    const throttled = throttle(fn, 1000);

    throttled('a');
    jest.advanceTimersByTime(1000);
    throttled('b');

    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith('b');
  });

  it('cancel suppresses a pending trailing call', () => {
    const fn = jest.fn();
    const throttled = throttle(fn, 1000);

    throttled('a');
    throttled('b');
    throttled.cancel();

    jest.advanceTimersByTime(1000);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenCalledWith('a');
  });
});
