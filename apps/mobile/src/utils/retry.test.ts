import { withRetry, HttpResponseError } from './retry';

// task_47c1 extension: the retry decision must key off HTTP status, not a
// lowercased substring match on the error message (which defaulted to
// retryable). A 4xx must never be retried even when its message matches none of
// the old prose literals; 5xx and network/abort errors must be.

describe('withRetry retryability', () => {
  it('does NOT retry an error carrying a 4xx status, whatever its message says', async () => {
    const err = Object.assign(new Error('the server said something bespoke'), { status: 422 });
    const fn = jest.fn(async () => {
      throw err;
    });

    await expect(withRetry(fn)).rejects.toBe(err);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('retries an error carrying a 5xx status', async () => {
    jest.useFakeTimers();
    try {
      const fn = jest.fn(async () => {
        throw new HttpResponseError(503);
      });

      const pending = withRetry(fn, { maxRetries: 2 });
      const assertion = expect(pending).rejects.toBeInstanceOf(HttpResponseError);
      await jest.runAllTimersAsync();
      await assertion;
      expect(fn).toHaveBeenCalledTimes(3); // 1 + 2 retries
    } finally {
      jest.useRealTimers();
    }
  });

  it('retries a plain network error', async () => {
    jest.useFakeTimers();
    try {
      const fn = jest.fn(async () => {
        throw new TypeError('Network request failed');
      });

      const pending = withRetry(fn, { maxRetries: 1 });
      const assertion = expect(pending).rejects.toBeInstanceOf(TypeError);
      await jest.runAllTimersAsync();
      await assertion;
      expect(fn).toHaveBeenCalledTimes(2);
    } finally {
      jest.useRealTimers();
    }
  });

  it('does NOT retry the client-side auth literals thrown before any response', async () => {
    for (const message of ['Session expired. Please log in again.', 'Not authenticated']) {
      const fn = jest.fn(async () => {
        throw new Error(message);
      });
      await expect(withRetry(fn)).rejects.toThrow(message);
      expect(fn).toHaveBeenCalledTimes(1);
    }
  });

  it('stops after maxRetries and rejects with the last error', async () => {
    jest.useFakeTimers();
    try {
      const fn = jest.fn(async () => {
        throw new HttpResponseError(500, 'attempt failed');
      });

      const pending = withRetry(fn, { maxRetries: 3 });
      const assertion = expect(pending).rejects.toThrow('attempt failed');
      await jest.runAllTimersAsync();
      await assertion;
      expect(fn).toHaveBeenCalledTimes(4);
    } finally {
      jest.useRealTimers();
    }
  });
});
