import { NetworkError } from './errors';

/**
 * fetch(), converting "couldn't reach the server" into a NetworkError so
 * callers can tell it apart from a rejected credential. Anything else is
 * rethrown untouched.
 *
 * An aborted fetch can reject with a DOMException, which does not extend
 * Error in every runtime, so the abort case is matched by name rather than
 * type.
 */
export async function fetchOrNetworkError(url: string, init?: RequestInit): Promise<Response> {
  try {
    return await fetch(url, init);
  } catch (error) {
    if (error instanceof TypeError || (error as { name?: string } | null)?.name === 'AbortError') {
      throw new NetworkError();
    }
    throw error;
  }
}
