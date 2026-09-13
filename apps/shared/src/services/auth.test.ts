import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  configureDefaultServerUrl,
  configureStorage,
  type StorageAdapter,
} from '../config/api';
import { HttpError, NetworkError } from '../utils/errors';
import { AuthService } from './auth';

// The shared auth layer is framework-agnostic: persistence arrives through an
// injected StorageAdapter, so these tests run in plain Node against an
// in-memory map rather than AsyncStorage or localStorage.
function inMemoryStorage(): StorageAdapter {
  const map = new Map<string, string>();
  return {
    getItem: (key) => Promise.resolve(map.get(key) ?? null),
    setItem: (key, value) => {
      map.set(key, value);
      return Promise.resolve();
    },
    removeItem: (key) => {
      map.delete(key);
      return Promise.resolve();
    },
  };
}

const jsonResponse = (status: number, body: unknown) =>
  ({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(JSON.stringify(body)),
  }) as Response;

const SESSION = {
  access_token: 'access-2',
  refresh_token: 'refresh-2',
  expires_in: 3600,
  user: {
    id: 'user-1',
    createdAt: '2020-01-01T00:00:00Z',
    updatedAt: '2020-01-01T00:00:00Z',
    lastLoginAt: '2020-01-01T00:00:00Z',
  },
};

beforeEach(async () => {
  configureStorage(inMemoryStorage());
  configureDefaultServerUrl(() => 'https://api.test');
  vi.spyOn(console, 'error').mockImplementation(() => {});
  vi.spyOn(console, 'log').mockImplementation(() => {});
  // Session state is module-level, so it survives between tests — reset it.
  await AuthService.clearTokens();
});

afterEach(() => {
  vi.restoreAllMocks();
});

/** A stored session whose access token is already past the refresh buffer. */
async function withExpiredToken() {
  await AuthService.saveTokens({
    accessToken: 'access-1',
    refreshToken: 'refresh-1',
    expiresAt: Date.now() - 1000,
  });
}

/** A stored session comfortably inside its validity window. */
async function withValidToken() {
  await AuthService.saveTokens({
    accessToken: 'access-1',
    refreshToken: 'refresh-1',
    expiresAt: Date.now() + 60 * 60 * 1000,
  });
}

// task_cab7 / task_f19d, previously mobile-only: a failed refresh may only clear
// tokens when the server actually rejected the credential. Before task_47c1 the
// web copy of this state machine cleared tokens on *any* refresh error, so a
// merely-offline browser was logged out; consolidating onto this implementation
// gives web the mobile semantics.
describe('token refresh: rejected vs unreachable', () => {
  it('keeps tokens and throws NetworkError when the request cannot reach the server', async () => {
    await withValidToken();
    global.fetch = vi.fn().mockRejectedValue(new TypeError('fetch failed'));

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('keeps tokens and throws NetworkError when the response body is unparseable', async () => {
    await withValidToken();
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.reject(new SyntaxError('Unexpected token <')),
    } as unknown as Response);

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('keeps tokens and throws NetworkError when the refresh 2xx is missing session fields', async () => {
    await withValidToken();
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(200, { data: { access_token: 'a' } }));

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
  });

  it.each([401, 403])('clears tokens when the server rejects the refresh with %i', async (status) => {
    await withValidToken();
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(status, { message: 'invalid refresh token' }));

    await expect(AuthService.refreshAccessToken()).rejects.toThrow('invalid refresh token');

    expect(await AuthService.getAccessToken()).toBeNull();
    expect(await AuthService.hasRefreshToken()).toBe(false);
  });

  // 400/404/429/5xx are not credential rejections: a rate-limited or broken
  // refresh endpoint must not log the user out.
  it.each([400, 404, 429, 500, 503])('keeps tokens on a %i refresh response', async (status) => {
    await withValidToken();
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(status, { message: 'nope' }));

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('stores the rotated token pair on a successful refresh', async () => {
    await withValidToken();
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(200, { data: SESSION }));

    await AuthService.refreshAccessToken();

    expect(await AuthService.getAccessToken()).toBe('access-2');
  });

  // H14: a refresh must never leak token material or the raw server body to logs.
  it('logs neither token material nor the response body on a failed refresh', async () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    await AuthService.saveTokens({
      accessToken: 'access-abcdef0123456789',
      refreshToken: 'refresh-fedcba9876543210',
      expiresAt: Date.now() + 60 * 60 * 1000,
    });
    global.fetch = vi
      .fn()
      .mockResolvedValue(jsonResponse(401, { message: 'nope', debug: 'SENSITIVE_MARKER' }));

    await expect(AuthService.refreshAccessToken()).rejects.toThrow();

    const logged = [...errorSpy.mock.calls, ...logSpy.mock.calls]
      .flat()
      .map((arg) => (typeof arg === 'string' ? arg : JSON.stringify(arg)))
      .join('\n');
    expect(logged).not.toContain('refresh-fedcba9876543210');
    expect(logged).not.toContain('access-abcdef0123456789');
    expect(logged).not.toContain('SENSITIVE_MARKER');
  });
});

describe('ensureValidToken', () => {
  it('propagates NetworkError so callers can tell offline from a dead session', async () => {
    await withExpiredToken();
    global.fetch = vi.fn().mockRejectedValue(new TypeError('fetch failed'));

    await expect(AuthService.ensureValidToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
  });

  it('returns false when the server rejects the refresh', async () => {
    await withExpiredToken();
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(401, { message: 'invalid refresh token' }));

    await expect(AuthService.ensureValidToken()).resolves.toBe(false);

    expect(await AuthService.getAccessToken()).toBeNull();
  });

  it('returns false without touching the network when there is no session', async () => {
    const fetchMock = vi.fn();
    global.fetch = fetchMock;

    await expect(AuthService.ensureValidToken()).resolves.toBe(false);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns true without refreshing while the token is still inside its window', async () => {
    await withValidToken();
    const fetchMock = vi.fn();
    global.fetch = fetchMock;

    await expect(AuthService.ensureValidToken()).resolves.toBe(true);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  // The refresh-dedup mutex: concurrent callers must share one in-flight refresh
  // rather than racing several and rotating the refresh token out from under
  // each other.
  it('dedupes concurrent refreshes into a single request', async () => {
    await withExpiredToken();
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { data: SESSION }));
    global.fetch = fetchMock;

    const results = await Promise.all([
      AuthService.ensureValidToken(),
      AuthService.ensureValidToken(),
      AuthService.ensureValidToken(),
    ]);

    expect(results).toEqual([true, true, true]);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe('fetchWithAuth', () => {
  it('throws the load-bearing session-expired message, without a request, when there is no session', async () => {
    const fetchMock = vi.fn();
    global.fetch = fetchMock;

    // Verbatim text: mobile's utils/retry.ts decides retryability by substring-
    // matching this message, so rephrasing it makes a dead session retryable.
    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toThrow(
      'Session expired. Please log in again.',
    );
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('attaches the bearer token and returns the response', async () => {
    await withValidToken();
    const ok = { status: 200, ok: true } as Response;
    const fetchMock = vi.fn().mockResolvedValue(ok);
    global.fetch = fetchMock;

    await expect(AuthService.fetchWithAuth('https://api.test/x')).resolves.toBe(ok);

    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect((init.headers as Headers).get('Authorization')).toBe('Bearer access-1');
  });

  it('merges caller headers supplied as a Headers instance and still owns Authorization', async () => {
    await withValidToken();
    const fetchMock = vi.fn().mockResolvedValue({ status: 200, ok: true } as Response);
    global.fetch = fetchMock;

    await AuthService.fetchWithAuth('https://api.test/x', {
      headers: new Headers({ 'X-Trace': 'abc', Authorization: 'Bearer attacker' }),
    });

    const headers = (fetchMock.mock.calls[0][1] as RequestInit).headers as Headers;
    expect(headers.get('X-Trace')).toBe('abc');
    expect(headers.get('Authorization')).toBe('Bearer access-1');
  });

  it('refreshes once on a 401 and retries with the rotated token', async () => {
    await withValidToken();
    const retried = { status: 200, ok: true } as Response;
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ status: 401, ok: false } as Response)
      .mockResolvedValueOnce(jsonResponse(200, { data: SESSION }))
      .mockResolvedValueOnce(retried);
    global.fetch = fetchMock;

    await expect(AuthService.fetchWithAuth('https://api.test/x')).resolves.toBe(retried);

    const retryInit = fetchMock.mock.calls[2][1] as RequestInit;
    expect((retryInit.headers as Headers).get('Authorization')).toBe('Bearer access-2');
  });

  it('clears tokens and throws session-expired when the 401 refresh is rejected', async () => {
    await withValidToken();
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce({ status: 401, ok: false } as Response)
      .mockResolvedValueOnce(jsonResponse(401, { message: 'invalid refresh token' }));

    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toThrow(
      'Session expired. Please log in again.',
    );
    expect(await AuthService.getAccessToken()).toBeNull();
  });

  it('keeps tokens and surfaces NetworkError when the 401 refresh cannot reach the server', async () => {
    await withValidToken();
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce({ status: 401, ok: false } as Response)
      .mockRejectedValueOnce(new TypeError('fetch failed'));

    await expect(AuthService.fetchWithAuth('https://api.test/x')).rejects.toBeInstanceOf(
      NetworkError,
    );
    expect(await AuthService.getAccessToken()).toBe('access-1');
  });

  // Characterization test for the known H12 gap, tracked as bug_8123: a retried
  // request that is *also* rejected is handed back to the caller as an ordinary
  // 401 response rather than ending the session. task_47c1 moved this behavior
  // without changing it, so both platforms now share the one copy to fix.
  // bug_8123 will invert this assertion — that is expected, not a regression.
  it('currently returns a second 401 to the caller instead of ending the session (bug_8123)', async () => {
    await withValidToken();
    const secondUnauthorized = { status: 401, ok: false } as Response;
    global.fetch = vi
      .fn()
      .mockResolvedValueOnce({ status: 401, ok: false } as Response)
      .mockResolvedValueOnce(jsonResponse(200, { data: SESSION }))
      .mockResolvedValueOnce(secondUnauthorized);

    await expect(AuthService.fetchWithAuth('https://api.test/x')).resolves.toBe(
      secondUnauthorized,
    );
    expect(await AuthService.getAccessToken()).toBe('access-2');
  });
});

describe('login and register', () => {
  it('stores the session returned by a successful email login', async () => {
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(200, { data: SESSION }));

    await AuthService.loginWithEmail({ email: 'a@b.com', password: 'pw' });

    expect(await AuthService.getAccessToken()).toBe('access-2');
    expect(await AuthService.getUserId()).toBe('user-1');
  });

  // task_f19d: callers need the status by type, so "server is broken" can be
  // told apart from "the credential was rejected" without parsing prose.
  it.each([401, 500])('throws HttpError carrying the %i status and server message', async (status) => {
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(status, { message: 'server says no' }));

    const error = await AuthService.loginWithEmail({ email: 'a@b.com', password: 'pw' }).catch(
      (e: unknown) => e,
    );

    expect(error).toBeInstanceOf(HttpError);
    expect((error as HttpError).status).toBe(status);
    expect((error as Error).message).toBe('server says no');
  });

  it('surfaces a captive-portal 200 that is not JSON as NetworkError, not a rejection', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve('<html>Please log in to the WiFi</html>'),
    } as unknown as Response);

    await expect(
      AuthService.registerWithEmail({ email: 'a@b.com', password: 'pw' }),
    ).rejects.toBeInstanceOf(NetworkError);
  });
});

describe('logout', () => {
  it('clears the session even when the logout call cannot reach the server', async () => {
    await withValidToken();
    global.fetch = vi.fn().mockRejectedValue(new TypeError('fetch failed'));

    await AuthService.logout();

    expect(await AuthService.getAccessToken()).toBeNull();
    expect(await AuthService.hasRefreshToken()).toBe(false);
  });

  it('notifies auth-state listeners that the session ended', async () => {
    await withValidToken();
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(200, {}));
    const listener = vi.fn();
    const unsubscribe = AuthService.onAuthStateChange(listener);

    await AuthService.logout();
    expect(listener).toHaveBeenCalledWith(false);

    unsubscribe();
    await AuthService.clearTokens();
    expect(listener).toHaveBeenCalledTimes(1);
  });
});

describe('changePassword', () => {
  it('accepts a 204 with an empty body', async () => {
    await withValidToken();
    await AuthService.saveUser(SESSION.user);
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      text: () => Promise.resolve(''),
    } as unknown as Response);

    await expect(AuthService.changePassword('old', 'new')).resolves.toBeUndefined();
  });

  it('reports the server message on failure', async () => {
    await withValidToken();
    await AuthService.saveUser(SESSION.user);
    global.fetch = vi.fn().mockResolvedValue(jsonResponse(400, { message: 'wrong password' }));

    await expect(AuthService.changePassword('old', 'new')).rejects.toThrow('wrong password');
  });
});
