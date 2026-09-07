import { AuthService } from './auth';
import { NetworkError } from '../utils/errors';

// chore_1089: every previously-bare fetch in auth.ts is now routed through
// the shared fetchOrNetworkError helper. These tests assert each newly
// wrapped path rejects with NetworkError (not a bare TypeError) when the
// server is unreachable, so AccountScreen et al. surface
// "Unable to reach the server..." instead of a raw "Network request failed".
// Each of these tests fails against pre-chore_1089 code, since the raw
// fetch()/TypeError would propagate unconverted.

jest.mock('expo-application', () => ({
  getIosIdForVendorAsync: jest.fn().mockResolvedValue('test-device-id'),
  getAndroidId: jest.fn().mockReturnValue('test-device-id'),
}));

const OFFLINE = () => jest.fn().mockRejectedValue(new TypeError('Network request failed')) as unknown as typeof fetch;

describe('AuthService network-error wrapping', () => {
  beforeEach(() => {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'log').mockImplementation(() => {});
  });

  afterEach(async () => {
    await AuthService.clearTokens();
    jest.restoreAllMocks();
  });

  it('loginWithDevice rejects with NetworkError when the server is unreachable', async () => {
    global.fetch = OFFLINE();
    await expect(AuthService.loginWithDevice()).rejects.toBeInstanceOf(NetworkError);
  });

  it('registerWithDevice rejects with NetworkError when the server is unreachable', async () => {
    global.fetch = OFFLINE();
    await expect(AuthService.registerWithDevice()).rejects.toBeInstanceOf(NetworkError);
  });

  it('loginWithEmail rejects with NetworkError when the server is unreachable', async () => {
    global.fetch = OFFLINE();
    await expect(
      AuthService.loginWithEmail({ email: 'a@example.com', password: 'pw' }),
    ).rejects.toBeInstanceOf(NetworkError);
  });

  it('registerWithEmail rejects with NetworkError when the server is unreachable', async () => {
    global.fetch = OFFLINE();
    await expect(
      AuthService.registerWithEmail({ email: 'a@example.com', password: 'pw' }),
    ).rejects.toBeInstanceOf(NetworkError);
  });

  // Regression guard, not a proof: logout() already swallowed this failure
  // before chore_1089 (it only logs and continues), so this passes both
  // before and after. Kept to pin that swallowing behavior now that the
  // error type flowing into the catch is NetworkError instead of TypeError.
  it('logout swallows the unreachable-server failure and still clears tokens', async () => {
    await AuthService.saveTokens({ accessToken: 'access-1', refreshToken: 'refresh-1' });
    global.fetch = OFFLINE();

    await AuthService.logout();

    expect(await AuthService.getAccessToken()).toBeNull();
    expect(await AuthService.hasRefreshToken()).toBe(false);
  });

  it('upgradeAccount rejects with NetworkError, not a bare TypeError, when the server is unreachable', async () => {
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      expiresAt: Date.now() + 60 * 60 * 1000,
    });
    await AuthService.saveUser({
      id: 'user-1',
      createdAt: '2020-01-01T00:00:00Z',
      updatedAt: '2020-01-01T00:00:00Z',
      lastLoginAt: '2020-01-01T00:00:00Z',
    });
    global.fetch = OFFLINE();

    await expect(AuthService.upgradeAccount('a@example.com', 'pw')).rejects.toBeInstanceOf(NetworkError);
  });

  it('changePassword rejects with NetworkError, not a bare TypeError, when the server is unreachable', async () => {
    await AuthService.saveTokens({
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
      expiresAt: Date.now() + 60 * 60 * 1000,
    });
    await AuthService.saveUser({
      id: 'user-1',
      createdAt: '2020-01-01T00:00:00Z',
      updatedAt: '2020-01-01T00:00:00Z',
      lastLoginAt: '2020-01-01T00:00:00Z',
    });
    global.fetch = OFFLINE();

    await expect(AuthService.changePassword('old-pw', 'new-pw')).rejects.toBeInstanceOf(NetworkError);
  });
});
