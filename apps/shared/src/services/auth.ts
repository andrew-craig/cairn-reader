// Shared authentication / token-refresh state machine for apps/web and
// apps/mobile (task_47c1). This used to be a near-verbatim copy in each app's
// services/auth.ts; the only real differences were the persistence backend
// (AsyncStorage vs localStorage) and the set of login flavors (device vs
// email). Persistence is injected via the same StorageAdapter the server-URL
// layer already uses (configureStorage), so nothing here knows which platform
// it is running on.
//
// apps/web re-exports this class unchanged. apps/mobile subclasses it to add
// the device-ID login flavors, upgradeAccount, and the retrying fetch wrapper.
//
// Session state lives in module-level variables rather than static class
// fields: a static field assigned through a subclass (`this.accessToken = x`
// inside a static method reached via the subclass) would create an own
// property on the subclass and shadow the base, silently splitting the
// session in two. Module-level state is a true singleton however it is
// reached.
import { getServerUrl, getStorage } from '../config/api';
import { HttpError, NetworkError } from '../utils/errors';
import { fetchOrNetworkError } from '../utils/http';
import type {
  AuthTokens,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  User,
} from '../types/auth';

// The same keys api.ts wipes on a server switch — see SESSION_STORAGE_KEYS there.
const ACCESS_TOKEN_KEY = '@cairn:access_token';
const REFRESH_TOKEN_KEY = '@cairn:refresh_token';
const USER_KEY = '@cairn:user';
const TOKEN_EXPIRES_AT_KEY = '@cairn:token_expires_at';

// Buffer before expiry that triggers a proactive refresh (5 minutes).
const TOKEN_EXPIRATION_BUFFER_MS = 5 * 60 * 1000;

type AuthStateListener = (isAuthenticated: boolean) => void;

let accessToken: string | null = null;
let refreshToken: string | null = null;
let currentUser: User | null = null;
let expiresAt: number | null = null;

let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;
const listeners: Set<AuthStateListener> = new Set();

export class AuthService {
  /**
   * Register a listener for auth state changes.
   * Returns an unsubscribe function.
   */
  static onAuthStateChange(listener: AuthStateListener): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }

  private static notifyListeners(isAuthenticated: boolean): void {
    listeners.forEach((listener) => {
      try {
        listener(isAuthenticated);
      } catch (error) {
        console.error('Error in auth state listener:', error);
      }
    });
  }

  /**
   * Parse a response body as JSON. A successful empty body (e.g. 204 No Content
   * from changePassword) is valid and yields {}. An unparseable non-empty body
   * means we did not actually reach the API — a captive portal answering 200
   * with an HTML login page, say — so that surfaces as NetworkError rather than
   * a definitive rejection.
   */
  protected static async parseJsonResponse(response: Response): Promise<unknown> {
    const text = await response.text();
    if (!text) {
      return {};
    }
    try {
      return JSON.parse(text);
    } catch {
      throw new NetworkError();
    }
  }

  static async initialize(): Promise<void> {
    const storage = getStorage();
    accessToken = await storage.getItem(ACCESS_TOKEN_KEY);
    refreshToken = await storage.getItem(REFRESH_TOKEN_KEY);
    const userJson = await storage.getItem(USER_KEY);
    currentUser = userJson ? (JSON.parse(userJson) as User) : null;
    const expiresAtStr = await storage.getItem(TOKEN_EXPIRES_AT_KEY);
    expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
  }

  /**
   * Shared POST-credentials-then-store flow behind every login/register flavor.
   * A non-2xx response throws HttpError carrying the status, so callers can
   * tell a rejected credential (4xx) from a broken server (5xx) without
   * parsing the message text (task_f19d).
   */
  protected static async authenticate(
    path: string,
    body: object,
    fallbackMessage: string,
  ): Promise<LoginResponse> {
    const response = await fetchOrNetworkError(`${getServerUrl()}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    const result = (await this.parseJsonResponse(response)) as {
      data?: LoginResponse;
      message?: string;
      error?: string;
    };

    if (!response.ok) {
      throw new HttpError(response.status, result.message || result.error || fallbackMessage);
    }

    const data = result.data;
    if (!data) {
      // 2xx with no session payload means we did not reach the real API.
      throw new NetworkError();
    }

    const tokenExpiresAt = Date.now() + data.expires_in * 1000;
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt: tokenExpiresAt,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async loginWithEmail(credentials: LoginRequest): Promise<LoginResponse> {
    return this.authenticate('/api/v1/auth/login', credentials, 'Email login failed');
  }

  static async registerWithEmail(credentials: RegisterRequest): Promise<LoginResponse> {
    return this.authenticate('/api/v1/auth/register', credentials, 'Email registration failed');
  }

  static async logout(): Promise<void> {
    if (refreshToken) {
      try {
        await fetchOrNetworkError(`${getServerUrl()}/api/v1/auth/logout`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${accessToken}`,
          },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
      } catch (error) {
        console.error('Error during logout:', error);
      }
    }

    await this.clearTokens();
  }

  static async saveTokens(tokens: AuthTokens): Promise<void> {
    accessToken = tokens.accessToken;
    refreshToken = tokens.refreshToken;
    expiresAt = tokens.expiresAt || null;

    const storage = getStorage();
    await storage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    await storage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    if (tokens.expiresAt) {
      await storage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString());
    }
  }

  static async clearTokens(): Promise<void> {
    accessToken = null;
    refreshToken = null;
    currentUser = null;
    expiresAt = null;

    const storage = getStorage();
    await storage.removeItem(ACCESS_TOKEN_KEY);
    await storage.removeItem(REFRESH_TOKEN_KEY);
    await storage.removeItem(USER_KEY);
    await storage.removeItem(TOKEN_EXPIRES_AT_KEY);

    // Notify listeners that user is no longer authenticated
    this.notifyListeners(false);
  }

  static async getAccessToken(): Promise<string | null> {
    if (!accessToken) {
      accessToken = await getStorage().getItem(ACCESS_TOKEN_KEY);
    }
    return accessToken;
  }

  static async isAuthenticated(): Promise<boolean> {
    const token = await this.getAccessToken();
    return token !== null;
  }

  /**
   * Check if the access token is expired or will expire soon.
   * Returns true if token should be refreshed.
   */
  static async isTokenExpired(): Promise<boolean> {
    if (!expiresAt) {
      const expiresAtStr = await getStorage().getItem(TOKEN_EXPIRES_AT_KEY);
      expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
    }

    // If no expiration stored, assume expired to trigger refresh
    if (!expiresAt) {
      return true;
    }

    // Consider expired if within buffer time of expiration
    return Date.now() >= expiresAt - TOKEN_EXPIRATION_BUFFER_MS;
  }

  /**
   * Check if we have a refresh token available.
   */
  static async hasRefreshToken(): Promise<boolean> {
    if (!refreshToken) {
      refreshToken = await getStorage().getItem(REFRESH_TOKEN_KEY);
    }
    return refreshToken !== null;
  }

  /**
   * Ensure we have a valid access token, refreshing if necessary.
   * This method handles concurrent refresh requests by reusing the same promise.
   * Returns true if a valid token is available, false if user needs to re-login.
   *
   * Throws NetworkError if the server could not be reached: "offline" is not
   * "please log in again", and callers must be able to tell them apart.
   */
  static async ensureValidToken(): Promise<boolean> {
    const hasToken = await this.isAuthenticated();
    if (!hasToken) {
      return false;
    }

    const isExpired = await this.isTokenExpired();
    if (!isExpired) {
      return true;
    }

    // Token is expired, need to refresh
    const hasRefresh = await this.hasRefreshToken();
    if (!hasRefresh) {
      await this.clearTokens();
      return false;
    }

    try {
      await this.refreshAccessToken();
      return true;
    } catch (error) {
      if (error instanceof NetworkError) {
        // Server unreachable — not a rejection. Let the caller distinguish
        // this from a genuine "please log in again" by propagating it.
        throw error;
      }
      console.error(
        '[Auth] Failed to refresh token:',
        error instanceof Error ? error.message : String(error),
      );
      return false;
    }
  }

  static async refreshAccessToken(): Promise<void> {
    // If already refreshing, wait for the existing refresh to complete
    if (isRefreshing && refreshPromise) {
      return refreshPromise;
    }

    isRefreshing = true;
    refreshPromise = this.doRefreshAccessToken();

    try {
      await refreshPromise;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  }

  private static async doRefreshAccessToken(): Promise<void> {
    if (!refreshToken) {
      refreshToken = await getStorage().getItem(REFRESH_TOKEN_KEY);
    }

    if (!refreshToken) {
      console.error('[Auth] No refresh token available in doRefreshAccessToken');
      throw new Error('No refresh token available');
    }

    let response: Response;
    try {
      // Not routed through fetchOrNetworkError: this catch converts *any*
      // thrown error to NetworkError (task_cab7), which is deliberately
      // broader than fetchOrNetworkError's TypeError/AbortError-only guard.
      // Losing a refresh to an unrecognized failure mode must never look like
      // a rejected credential.
      response = await fetch(`${getServerUrl()}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch (error) {
      // Could not reach the server at all (offline, timeout, DNS, etc.). This is
      // not a rejection of the credential, so the refresh token must survive.
      console.error(
        '[Auth] Refresh request failed, server unreachable:',
        error instanceof Error ? error.message : String(error),
      );
      throw new NetworkError();
    }

    let result;
    try {
      result = await response.json();
    } catch (parseError) {
      console.error('[Auth] Failed to parse refresh response as JSON:', parseError);
      throw new NetworkError();
    }

    if (!response.ok) {
      console.error('[Auth] Refresh failed with error response, status:', response.status);

      // Only 401 (invalid/expired/reused refresh token, per the refresh
      // endpoint's documented responses) or 403 (a definitive authorization
      // rejection, e.g. a disabled account) means the server rejected the
      // credential. Every other status — including 400 (malformed request),
      // 404, and 429 (rate limited; /auth/* is rate-limited per IP) — is not
      // a credential rejection and must not log the user out.
      if (response.status !== 401 && response.status !== 403) {
        throw new NetworkError();
      }

      console.error('[Auth] Refresh rejected by server, clearing tokens');
      await this.clearTokens();
      throw new Error(result.message || result.error || 'Failed to refresh token');
    }

    const data: LoginResponse = result.data;

    if (!data || !data.access_token || !data.refresh_token || !data.expires_in) {
      console.error('[Auth] Refresh response missing required fields');
      throw new NetworkError();
    }

    const tokenExpiresAt = Date.now() + data.expires_in * 1000;

    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt: tokenExpiresAt,
    });
  }

  static async saveUser(user: User): Promise<void> {
    currentUser = user;
    await getStorage().setItem(USER_KEY, JSON.stringify(user));
  }

  static async getUser(): Promise<User | null> {
    if (!currentUser) {
      const userJson = await getStorage().getItem(USER_KEY);
      currentUser = userJson ? (JSON.parse(userJson) as User) : null;
    }
    return currentUser;
  }

  static async getUserId(): Promise<string | null> {
    const user = await this.getUser();
    return user?.id || null;
  }

  /**
   * Authenticated fetch with proactive token refresh and a single reactive 401
   * retry. Feature services (read, explore) build on this — one implementation,
   * not a private copy per service.
   *
   * The 'Session expired. Please log in again.' and 'Not authenticated' error
   * messages are load-bearing: mobile's utils/retry.ts classifies retryability
   * by substring-matching the message, so rephrasing either makes an
   * unrecoverable auth failure retryable.
   */
  static async fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
    // Proactively check and refresh token if expired before making request
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const token = await this.getAccessToken();

    if (!token) {
      throw new Error('Not authenticated');
    }

    // Build headers via the Headers API so any valid HeadersInit (plain object,
    // Headers instance, or entry array) from callers merges correctly. We always
    // own Authorization; callers may still set their own Content-Type.
    const buildHeaders = (bearer: string): Headers => {
      const headers = new Headers(options.headers);
      if (!headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json');
      }
      headers.set('Authorization', `Bearer ${bearer}`);
      return headers;
    };

    const response = await fetchOrNetworkError(url, {
      ...options,
      headers: buildHeaders(token),
    });

    // Handle 401 Unauthorized - try to refresh token (fallback for edge cases)
    if (response.status === 401) {
      try {
        await this.refreshAccessToken();
        const newAccessToken = await this.getAccessToken();

        // Retry the request with new token
        return await fetchOrNetworkError(url, {
          ...options,
          headers: buildHeaders(newAccessToken ?? ''),
        });
      } catch (error) {
        if (error instanceof NetworkError) {
          // Server unreachable — not a rejection. Keep tokens and let the
          // caller retry later instead of forcing a logout.
          throw error;
        }
        // Refresh was rejected (or unavailable) for a real reason; ensure
        // tokens are cleared before forcing a logout.
        await this.clearTokens();
        throw new Error('Session expired. Please log in again.');
      }
    }

    return response;
  }

  static async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    const userId = await this.getUserId();
    if (!userId) {
      throw new Error('No user found');
    }

    const response = await this.fetchWithAuth(`${getServerUrl()}/api/v1/user/${userId}/password`, {
      method: 'PUT',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });

    const result = (await this.parseJsonResponse(response)) as {
      message?: string;
      error?: string;
    };

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to change password');
    }
  }
}
