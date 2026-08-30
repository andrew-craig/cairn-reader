import { AuthService } from './auth';
import { withRetry } from '../utils/retry';

// task_ca2d: mobile's read.ts and explore.ts each carried a private copy of the
// authenticated-fetch policy. These are now one implementation on AuthService.
// The tests below characterize the canonical behavior the two callers relied on
// and pin the load-bearing error-message-to-retryability contract.

describe('withRetry auth-error classification (load-bearing message contract)', () => {
  it('does NOT retry when the request throws "Session expired. Please log in again."', async () => {
    const fn = jest.fn(async () => {
      throw new Error('Session expired. Please log in again.');
    });

    await expect(withRetry(fn)).rejects.toThrow('Session expired. Please log in again.');
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it('does NOT retry when the request throws "Not authenticated"', async () => {
    const fn = jest.fn(async () => {
      throw new Error('Not authenticated');
    });

    await expect(withRetry(fn)).rejects.toThrow('Not authenticated');
    expect(fn).toHaveBeenCalledTimes(1);
  });
});

describe('AuthService.fetchWithAuth', () => {
  beforeEach(async () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'log').mockImplementation(() => {});
    // Valid, non-expiring session by default.
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      expiresAt: Date.now() + 60 * 60 * 1000,
    });
  });

  afterEach(async () => {
    await AuthService.clearTokens();
    jest.restoreAllMocks();
  });

  it('throws "Session expired. Please log in again." when no valid token can be obtained', async () => {
    await AuthService.clearTokens();
    global.fetch = jest.fn() as unknown as typeof fetch;

    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toThrow(
      'Session expired. Please log in again.',
    );
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it('attaches the bearer token and returns the response on success', async () => {
    const ok = { status: 200, ok: true } as Response;
    global.fetch = jest.fn().mockResolvedValue(ok) as unknown as typeof fetch;

    const res = await AuthService.fetchWithAuth('https://api.test/x');

    expect(res).toBe(ok);
    const [, init] = (global.fetch as jest.Mock).mock.calls[0];
    expect((init.headers as Headers).get('Authorization')).toBe('Bearer access-1');
  });

  it('refreshes once on 401 and retries the request with the new token', async () => {
    const unauthorized = { status: 401, ok: false } as Response;
    const retried = { status: 200, ok: true } as Response;
    const fetchMock = jest
      .fn()
      .mockResolvedValueOnce(unauthorized) // initial request
      .mockResolvedValueOnce({
        // refresh call
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            data: { access_token: 'access-2', refresh_token: 'refresh-2', expires_in: 3600 },
          }),
      })
      .mockResolvedValueOnce(retried); // retried request
    global.fetch = fetchMock as unknown as typeof fetch;

    const res = await AuthService.fetchWithAuth('https://api.test/x');

    expect(res).toBe(retried);
    const retryInit = fetchMock.mock.calls[2][1];
    expect((retryInit.headers as Headers).get('Authorization')).toBe('Bearer access-2');
  });

  it('clears tokens and throws the session-expired message when the 401 refresh fails', async () => {
    const unauthorized = { status: 401, ok: false } as Response;
    const fetchMock = jest
      .fn()
      .mockResolvedValueOnce(unauthorized)
      .mockResolvedValueOnce({
        ok: false,
        status: 401,
        text: async () => JSON.stringify({ message: 'bad refresh' }),
      });
    global.fetch = fetchMock as unknown as typeof fetch;

    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toThrow(
      'Session expired. Please log in again.',
    );
    expect(await AuthService.getAccessToken()).toBeNull();
  });
});

describe('AuthService.fetchWithAuthAndRetry', () => {
  beforeEach(async () => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'log').mockImplementation(() => {});
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      expiresAt: Date.now() + 60 * 60 * 1000,
    });
  });

  afterEach(async () => {
    await AuthService.clearTokens();
    jest.restoreAllMocks();
  });

  it('returns 4xx responses as-is without retrying', async () => {
    const notFound = { status: 404, ok: false } as Response;
    global.fetch = jest.fn().mockResolvedValue(notFound) as unknown as typeof fetch;

    const res = await AuthService.fetchWithAuthAndRetry('https://api.test/x');

    expect(res).toBe(notFound);
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('throws "Server error <status>" on a 5xx so withRetry retries and eventually rejects', async () => {
    jest.useFakeTimers();
    try {
      global.fetch = jest
        .fn()
        .mockResolvedValue({ status: 503, ok: false }) as unknown as typeof fetch;

      const pending = AuthService.fetchWithAuthAndRetry('https://api.test/x');
      const assertion = expect(pending).rejects.toThrow('Server error 503');
      await jest.runAllTimersAsync();
      await assertion;
      // 1 initial + 3 retries
      expect((global.fetch as jest.Mock).mock.calls.length).toBe(4);
    } finally {
      jest.useRealTimers();
    }
  });
});
