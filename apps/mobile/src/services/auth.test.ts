import { AuthService } from './auth';
import { NetworkError, HttpError } from '@cairn/shared';

jest.mock('expo-application', () => ({
  getIosIdForVendorAsync: jest.fn().mockResolvedValue('test-device-id'),
  getAndroidId: jest.fn().mockReturnValue('test-device-id'),
}));

// H14: token refresh must never leak token material (raw token, token
// prefix/preview, or full response bodies) to device logs, and diagnostic
// logging must be gated behind __DEV__.
describe('AuthService token refresh logging', () => {
  const OLD_REFRESH_TOKEN = 'old-refresh-token-0123456789abcdef-secret';
  const NEW_REFRESH_TOKEN = 'new-refresh-token-fedcba9876543210-secret';
  const NEW_ACCESS_TOKEN = 'new-access-token-aaaabbbbccccdddd-secret';
  const originalDev = (global as unknown as { __DEV__: boolean }).__DEV__;

  let logSpy: jest.SpyInstance;
  let errorSpy: jest.SpyInstance;

  beforeEach(async () => {
    logSpy = jest.spyOn(console, 'log').mockImplementation(() => {});
    errorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
    await AuthService.saveTokens({
      accessToken: 'old-access-token',
      refreshToken: OLD_REFRESH_TOKEN,
    });
  });

  afterEach(() => {
    logSpy.mockRestore();
    errorSpy.mockRestore();
    (global as unknown as { __DEV__: boolean }).__DEV__ = originalDev;
    jest.restoreAllMocks();
  });

  function loggedText(): string {
    return [...logSpy.mock.calls, ...errorSpy.mock.calls]
      .flat()
      .map((arg) => (typeof arg === 'string' ? arg : JSON.stringify(arg)))
      .join('\n');
  }

  it('does not log the raw token or a token prefix on a successful refresh', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          access_token: NEW_ACCESS_TOKEN,
          refresh_token: NEW_REFRESH_TOKEN,
          expires_in: 3600,
        },
      }),
    }) as unknown as typeof fetch;

    await AuthService.refreshAccessToken();

    const text = loggedText();
    expect(text).not.toContain(OLD_REFRESH_TOKEN);
    expect(text).not.toContain(OLD_REFRESH_TOKEN.substring(0, 20));
    expect(text).not.toContain(NEW_REFRESH_TOKEN);
    expect(text).not.toContain(NEW_REFRESH_TOKEN.substring(0, 20));
    expect(text).not.toContain(NEW_ACCESS_TOKEN);
    expect(text).not.toContain(NEW_ACCESS_TOKEN.substring(0, 20));
  });

  it('does not log the token or the full response body on a failed refresh', async () => {
    const secretMarker = 'SENSITIVE_SERVER_DETAIL_MARKER';
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ message: 'invalid refresh token', debug: secretMarker }),
    }) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toThrow();

    const text = loggedText();
    expect(text).not.toContain(OLD_REFRESH_TOKEN);
    expect(text).not.toContain(OLD_REFRESH_TOKEN.substring(0, 20));
    expect(text).not.toContain(secretMarker);
  });

  it('suppresses diagnostic console.log output when __DEV__ is false', async () => {
    (global as unknown as { __DEV__: boolean }).__DEV__ = false;
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          access_token: 'access-2',
          refresh_token: 'refresh-2',
          expires_in: 3600,
        },
      }),
    }) as unknown as typeof fetch;

    await AuthService.refreshAccessToken();

    expect(logSpy).not.toHaveBeenCalled();
  });
});

// task_cab7: a failed refresh must only clear tokens when the server actually
// rejected the credential (4xx). Network errors, timeouts, 5xx responses, and
// malformed bodies must keep the tokens and surface a retryable NetworkError,
// otherwise a merely-offline device gets logged out.
describe('AuthService token refresh: offline vs rejected (task_cab7)', () => {
  beforeEach(async () => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('keeps tokens and throws a NetworkError when the refresh request cannot reach the server', async () => {
    global.fetch = jest.fn().mockRejectedValue(new TypeError('Network request failed')) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('clears tokens when the refresh is rejected with a 401', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ message: 'invalid refresh token' }),
    }) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toThrow('invalid refresh token');

    expect(await AuthService.getAccessToken()).toBeNull();
    expect(await AuthService.hasRefreshToken()).toBe(false);
  });

  it('keeps tokens and throws a NetworkError when the refresh endpoint returns a 500', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({ message: 'internal error' }),
    }) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('keeps tokens and throws a NetworkError when the refresh endpoint returns a 429 (rate limited)', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 429,
      json: async () => ({ message: 'rate limited' }),
    }) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('keeps tokens and throws a NetworkError when the refresh endpoint returns a 400 (malformed request)', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ message: 'invalid input' }),
    }) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
    expect(await AuthService.hasRefreshToken()).toBe(true);
  });

  it('ensureValidToken propagates a NetworkError when the server is unreachable (expired token)', async () => {
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      expiresAt: Date.now() - 1000,
    });
    global.fetch = jest.fn().mockRejectedValue(new TypeError('Network request failed')) as unknown as typeof fetch;

    await expect(AuthService.ensureValidToken()).rejects.toBeInstanceOf(NetworkError);

    expect(await AuthService.getAccessToken()).toBe('access-1');
  });

  it('ensureValidToken returns false when the server rejects the refresh (expired token)', async () => {
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      expiresAt: Date.now() - 1000,
    });
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ message: 'invalid refresh token' }),
    }) as unknown as typeof fetch;

    await expect(AuthService.ensureValidToken()).resolves.toBe(false);

    expect(await AuthService.getAccessToken()).toBeNull();
  });
});

// task_f19d: a 5xx auth response must be distinguishable, by type, from a 4xx
// credential rejection, so LoginScreen (and any future caller) can tell
// "server is broken" apart from "the credential was rejected" without
// parsing the message text. These tests fail against pre-change code: the
// four entry points below used to throw a plain Error for every non-2xx
// status, so `toBeInstanceOf(HttpError)` fails regardless of status.
describe('AuthService auth entry points: non-2xx responses throw HttpError (task_f19d)', () => {
  beforeEach(() => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  function mockJsonResponse(status: number, body: Record<string, unknown>) {
    global.fetch = jest.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      text: () => Promise.resolve(JSON.stringify(body)),
    }) as unknown as typeof fetch;
  }

  const entryPoints: [string, () => Promise<unknown>][] = [
    ['loginWithDevice', () => AuthService.loginWithDevice()],
    ['registerWithDevice', () => AuthService.registerWithDevice()],
    ['loginWithEmail', () => AuthService.loginWithEmail({ email: 'a@b.com', password: 'x' })],
    ['registerWithEmail', () => AuthService.registerWithEmail({ email: 'a@b.com', password: 'x' })],
  ];

  it.each(entryPoints)('%s throws an HttpError carrying the status and server message on a 500', async (_name, call) => {
    mockJsonResponse(500, { message: 'internal error' });

    const error = await call().catch((e) => e);

    expect(error).toBeInstanceOf(HttpError);
    expect((error as HttpError).status).toBe(500);
    expect((error as Error).message).toBe('internal error');
  });

  it.each(entryPoints)('%s throws an HttpError carrying the status and server message on a 401 (credential semantics unchanged)', async (_name, call) => {
    mockJsonResponse(401, { message: 'invalid credentials' });

    const error = await call().catch((e) => e);

    expect(error).toBeInstanceOf(HttpError);
    expect((error as HttpError).status).toBe(401);
    expect((error as Error).message).toBe('invalid credentials');
  });
});
