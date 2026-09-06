import { SystemService } from './system';
import { NetworkError } from '../utils/errors';

// chore_1089: system.ts's unauthenticated health check was one of the 9 raw
// fetch call sites; it now shares fetchOrNetworkError with AuthService so an
// unreachable server surfaces the same distinguishable NetworkError.
describe('SystemService.getServerVersion', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('returns the reported version on success', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ version: '1.2.3' }),
    }) as unknown as typeof fetch;

    await expect(SystemService.getServerVersion()).resolves.toBe('1.2.3');
  });

  it('rejects with a NetworkError, not a bare TypeError, when the server is unreachable', async () => {
    global.fetch = jest.fn().mockRejectedValue(new TypeError('Network request failed')) as unknown as typeof fetch;

    await expect(SystemService.getServerVersion()).rejects.toBeInstanceOf(NetworkError);
  });
});
