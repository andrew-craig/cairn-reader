import { fetchOrNetworkError } from './http';
import { NetworkError } from './errors';

// chore_1089: fetchOrNetworkError is the single place every service call site
// routes through to convert "couldn't reach the server" into a NetworkError,
// so callers can tell it apart from a rejected credential (4xx).
describe('fetchOrNetworkError', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('returns the response on success without touching it', async () => {
    const ok = { status: 200, ok: true } as Response;
    global.fetch = jest.fn().mockResolvedValue(ok) as unknown as typeof fetch;

    await expect(fetchOrNetworkError('https://api.test/x')).resolves.toBe(ok);
  });

  it('converts a TypeError (offline/DNS/connection-refused) into a NetworkError', async () => {
    global.fetch = jest.fn().mockRejectedValue(new TypeError('Network request failed')) as unknown as typeof fetch;

    await expect(fetchOrNetworkError('https://api.test/x')).rejects.toBeInstanceOf(NetworkError);
  });

  it('converts a plain-Error AbortError (timeout) into a NetworkError', async () => {
    const abortError = new Error('Aborted');
    abortError.name = 'AbortError';
    global.fetch = jest.fn().mockRejectedValue(abortError) as unknown as typeof fetch;

    await expect(fetchOrNetworkError('https://api.test/x')).rejects.toBeInstanceOf(NetworkError);
  });

  // task_c87c review fix (fourth instance of this bug class): a DOMException
  // does not extend Error in every runtime, so the guard must match by name,
  // not by `instanceof Error`.
  it('converts a non-Error DOMException AbortError (timeout) into a NetworkError', async () => {
    const abortError = new DOMException('Aborted', 'AbortError');
    expect(abortError).not.toBeInstanceOf(Error);
    global.fetch = jest.fn().mockRejectedValue(abortError) as unknown as typeof fetch;

    await expect(fetchOrNetworkError('https://api.test/x')).rejects.toBeInstanceOf(NetworkError);
  });

  it('rethrows any other error untouched', async () => {
    const other = new RangeError('something else entirely');
    global.fetch = jest.fn().mockRejectedValue(other) as unknown as typeof fetch;

    await expect(fetchOrNetworkError('https://api.test/x')).rejects.toBe(other);
  });
});
