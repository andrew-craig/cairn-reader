import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  AuthService,
  RefreshNetworkError,
  RefreshRejectedError,
} from './auth';
import {
  configureDefaultServerUrl,
  configureStorage,
  type StorageAdapter,
} from '../config/api';

// In-memory StorageAdapter — the injectable-adapter pattern the app wires to
// AsyncStorage (mobile) / localStorage (web) at startup.
function makeMemoryStorage(): StorageAdapter & { data: Map<string, string> } {
  const data = new Map<string, string>();
  return {
    data,
    async getItem(key) {
      return data.has(key) ? (data.get(key) as string) : null;
    },
    async setItem(key, value) {
      data.set(key, value);
    },
    async removeItem(key) {
      data.delete(key);
    },
  };
}

let storage: ReturnType<typeof makeMemoryStorage>;

const ACCESS_KEY = '@cairn:access_token';
const REFRESH_KEY = '@cairn:refresh_token';

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function bareResponse(status: number): Response {
  return { ok: status >= 200 && status < 300, status, text: async () => '' } as unknown as Response;
}

async function seedValidSession(expiresInMs = 60 * 60 * 1000) {
  await AuthService.saveTokens({
    accessToken: 'access-1',
    refreshToken: 'refresh-1',
    expiresAt: Date.now() + expiresInMs,
  });
}

beforeEach(async () => {
  storage = makeMemoryStorage();
  configureStorage(storage);
  configureDefaultServerUrl(() => 'https://api.test');
  await AuthService.clearTokens();
  vi.restoreAllMocks();
  vi.spyOn(console, 'error').mockImplementation(() => {});
  vi.spyOn(console, 'log').mockImplementation(() => {});
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('doRefreshAccessToken — offline must NOT clear tokens (network-failure-clears-tokens bug)', () => {
  it('keeps tokens when the refresh request cannot reach the server', async () => {
    await seedValidSession();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new TypeError('Network request failed')),
    );

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(RefreshNetworkError);

    // Tokens survive so a later attempt (once back online) can succeed.
    expect(storage.data.get(ACCESS_KEY)).toBe('access-1');
    expect(storage.data.get(REFRESH_KEY)).toBe('refresh-1');
  });

  it('keeps tokens when the server returns an unparseable 2xx body', async () => {
    await seedValidSession();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, status: 200, text: async () => '<html>gateway</html>' }),
    );

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(RefreshNetworkError);
    expect(storage.data.get(REFRESH_KEY)).toBe('refresh-1');
  });

  it('CLEARS tokens when the server actively rejects the refresh token', async () => {
    await seedValidSession();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(401, { message: 'invalid refresh token' })),
    );

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(RefreshRejectedError);
    expect(storage.data.get(ACCESS_KEY)).toBeUndefined();
    expect(storage.data.get(REFRESH_KEY)).toBeUndefined();
  });

  it('does not log token material on a failed refresh', async () => {
    await seedValidSession();
    const secret = 'SENSITIVE_SERVER_DETAIL';
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(401, { message: 'nope', debug: secret })),
    );

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(RefreshRejectedError);

    const logged = [
      ...(console.error as unknown as ReturnType<typeof vi.fn>).mock.calls,
      ...(console.log as unknown as ReturnType<typeof vi.fn>).mock.calls,
    ]
      .flat()
      .map((a) => (typeof a === 'string' ? a : JSON.stringify(a)))
      .join('\n');
    expect(logged).not.toContain('refresh-1');
    expect(logged).not.toContain(secret);
  });
});

describe('ensureValidToken', () => {
  it('returns true and keeps the session when a proactive refresh fails offline', async () => {
    await seedValidSession(-1000); // already past the refresh buffer
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('offline')));

    await expect(AuthService.ensureValidToken()).resolves.toBe(true);
    expect(storage.data.get(REFRESH_KEY)).toBe('refresh-1');
  });

  it('returns false when the server rejects the refresh token', async () => {
    await seedValidSession(-1000);
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(401, { message: 'nope' })));

    await expect(AuthService.ensureValidToken()).resolves.toBe(false);
  });

  it('dedupes concurrent refreshes into one request', async () => {
    await seedValidSession(-1000);
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse(200, {
          data: { access_token: 'access-2', refresh_token: 'refresh-2', expires_in: 3600 },
        }),
      );
    vi.stubGlobal('fetch', fetchMock);

    await Promise.all([
      AuthService.ensureValidToken(),
      AuthService.ensureValidToken(),
      AuthService.ensureValidToken(),
    ]);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(storage.data.get(ACCESS_KEY)).toBe('access-2');
  });
});

describe('fetchWithAuth', () => {
  it('attaches the bearer token', async () => {
    await seedValidSession();
    const fetchMock = vi.fn().mockResolvedValue(bareResponse(200));
    vi.stubGlobal('fetch', fetchMock);

    await AuthService.fetchWithAuth('https://api.test/x');

    const headers = fetchMock.mock.calls[0][1].headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer access-1');
  });

  it('refreshes once on 401 then retries with the new token', async () => {
    await seedValidSession();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(bareResponse(401))
      .mockResolvedValueOnce(
        jsonResponse(200, {
          data: { access_token: 'access-2', refresh_token: 'refresh-2', expires_in: 3600 },
        }),
      )
      .mockResolvedValueOnce(bareResponse(200));
    vi.stubGlobal('fetch', fetchMock);

    const res = await AuthService.fetchWithAuth('https://api.test/x');

    expect(res.status).toBe(200);
    const retryHeaders = fetchMock.mock.calls[2][1].headers as Headers;
    expect(retryHeaders.get('Authorization')).toBe('Bearer access-2');
  });

  it('H12: logs the user out on a SECOND 401 after a successful refresh', async () => {
    await seedValidSession();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(bareResponse(401)) // initial
      .mockResolvedValueOnce(
        jsonResponse(200, {
          data: { access_token: 'access-2', refresh_token: 'refresh-2', expires_in: 3600 },
        }),
      ) // refresh succeeds
      .mockResolvedValueOnce(bareResponse(401)); // retry still 401
    vi.stubGlobal('fetch', fetchMock);

    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toThrow(
      'Session expired. Please log in again.',
    );
    expect(storage.data.get(ACCESS_KEY)).toBeUndefined();
    expect(storage.data.get(REFRESH_KEY)).toBeUndefined();
  });

  it('on 401 + offline refresh: keeps tokens and throws a retryable (non-auth) error', async () => {
    await seedValidSession();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(bareResponse(401))
      .mockRejectedValueOnce(new TypeError('offline'));
    vi.stubGlobal('fetch', fetchMock);

    let err: unknown;
    try {
      await AuthService.fetchWithAuth('https://api.test/x');
    } catch (e) {
      err = e;
    }

    expect(err).toBeInstanceOf(RefreshNetworkError);
    // Message must not match retry.ts's non-retryable substrings.
    expect((err as Error).message.toLowerCase()).not.toMatch(
      /not authenticated|session expired|unauthorized|forbidden|not found|bad request/,
    );
    expect(storage.data.get(REFRESH_KEY)).toBe('refresh-1');
  });

  it('throws "Session expired. Please log in again." when there is no session', async () => {
    vi.stubGlobal('fetch', vi.fn());
    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toThrow(
      'Session expired. Please log in again.',
    );
    expect(fetch).not.toHaveBeenCalled();
  });
});
