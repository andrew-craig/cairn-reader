export interface Throttled<TArgs extends unknown[]> {
  (...args: TArgs): void;
  cancel: () => void;
}

// Leading + trailing throttle: fires immediately if idle, then coalesces any
// calls that arrive within the interval into a single trailing call carrying
// the latest args once the interval elapses.
export function throttle<TArgs extends unknown[]>(
  fn: (...args: TArgs) => void,
  intervalMs: number,
): Throttled<TArgs> {
  let lastCallTime = 0;
  let timer: ReturnType<typeof setTimeout> | null = null;
  let pendingArgs: TArgs | null = null;

  const throttled = (...args: TArgs) => {
    const now = Date.now();
    const elapsed = now - lastCallTime;

    if (elapsed >= intervalMs) {
      lastCallTime = now;
      fn(...args);
      return;
    }

    pendingArgs = args;
    if (timer) return;
    timer = setTimeout(() => {
      timer = null;
      lastCallTime = Date.now();
      const argsToSend = pendingArgs;
      pendingArgs = null;
      if (argsToSend) fn(...argsToSend);
    }, intervalMs - elapsed);
  };

  throttled.cancel = () => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    pendingArgs = null;
  };

  return throttled;
}
