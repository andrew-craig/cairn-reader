import { AuthService } from './auth';

// H14: token refresh must never leak token material (raw token, token
// prefix/preview, or full response bodies) to device logs. The state machine
// now lives in @cairn/shared; these tests pin that contract from the mobile
// side (mobile re-exports the shared AuthService via a subclass).
describe('AuthService token refresh logging', () => {
  const OLD_REFRESH_TOKEN = 'old-refresh-token-0123456789abcdef-secret';
  const NEW_REFRESH_TOKEN = 'new-refresh-token-fedcba9876543210-secret';
  const NEW_ACCESS_TOKEN = 'new-access-token-aaaabbbbccccdddd-secret';

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

  afterEach(async () => {
    await AuthService.clearTokens();
    logSpy.mockRestore();
    errorSpy.mockRestore();
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
      text: async () =>
        JSON.stringify({
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
      text: async () =>
        JSON.stringify({ message: 'invalid refresh token', debug: secretMarker }),
    }) as unknown as typeof fetch;

    await expect(AuthService.refreshAccessToken()).rejects.toThrow();

    const text = loggedText();
    expect(text).not.toContain(OLD_REFRESH_TOKEN);
    expect(text).not.toContain(OLD_REFRESH_TOKEN.substring(0, 20));
    expect(text).not.toContain(secretMarker);
  });
});
