import { getServerUrl } from '@cairn/shared';
import { fetchOrNetworkError } from '../utils/http';

/**
 * SystemService exposes backend metadata that isn't tied to a user session.
 */
export class SystemService {
  /**
   * Fetch the backend's reported version from the unauthenticated
   * /health/live endpoint. Returns null when the server doesn't report a
   * version (e.g. a deployment that predates version embedding).
   */
  static async getServerVersion(): Promise<string | null> {
    const response = await fetchOrNetworkError(`${getServerUrl()}/health/live`);
    if (!response.ok) {
      throw new Error('Failed to fetch server version');
    }
    const data = await response.json();
    return data && typeof data === 'object' && typeof data.version === 'string' ? data.version : null;
  }
}
