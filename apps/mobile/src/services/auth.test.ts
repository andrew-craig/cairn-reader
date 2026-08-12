import { AuthService } from './auth';

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
