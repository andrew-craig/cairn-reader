// Shared authentication / token-refresh state machine for apps/web and
// apps/mobile. Previously this was a near-verbatim copy in each app's
// services/auth.ts; the only real differences were the persistence backend
// (AsyncStorage vs localStorage) and the set of login flavors (device vs
// email). Persistence is injected via the same StorageAdapter the server-URL
// layer already uses (configureStorage); platform-specific login methods are
// added by subclassing (see apps/mobile/src/services/auth.ts). apps/web
// re-exports this class unchanged.
//
// Session state lives in module-level variables (a process-wide singleton) so
// it is shared regardless of which subclass a caller reaches it through.
import { getServerUrl, getStorage } from '../config/api';
import type {
  AuthTokens,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  User,
} from '../types/auth';

// These are the same keys api.ts wipes on a server switch (SESSION_STORAGE_KEYS).
const ACCESS_TOKEN_KEY = '@cairn:access_token';
const REFRESH_TOKEN_KEY = '@cairn:refresh_token';
const USER_KEY = '@cairn:user';
const TOKEN_EXPIRES_AT_KEY = '@cairn:token_expires_at';

// Buffer before expiry that triggers a proactive refresh (5 minutes).
const TOKEN_EXPIRATION_BUFFER_MS = 5 * 60 * 1000;

type AuthStateListener = (isAuthenticated: boolean) => void;

/** The users service actively rejected the refresh token — the session is dead. */
export class RefreshRejectedError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'RefreshRejectedError';
  }
}

/**
 * The refresh request could not reach the users service (offline, DNS failure,
 * timeout, unparseable/malformed response). The session may still be valid —
 * callers must NOT clear tokens for this.
 */
export class RefreshNetworkError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'RefreshNetworkError';
  }
}

let accessToken: string | null = null;
let refreshToken: string | null = null;
let currentUser: User | null = null;
let expiresAt: number | null = null;

let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;
const listeners: Set<AuthStateListener> = new Set();

export class AuthService {
  /** Register a listener for auth-state changes. Returns an unsubscribe fn. */
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

  protected static async parseJsonResponse(response: Response): Promise<unknown> {
    const text = await response.text();
    // A successful empty body (e.g. 204 No Content) is valid; only treat an
    // unparseable non-empty body as a server-unreachable error.
    if (!text) {
      return {};
    }
    try {
      return JSON.parse(text);
    } catch {
      throw new Error('Unable to reach the server. Please try again later.');
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

  /** Shared POST-credentials-and-store flow for every login/register flavor. */
  protected static async authenticate(path: string, body: object): Promise<LoginResponse> {
    const response = await fetch(`${getServerUrl()}${path}`, {
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
      throw new Error(result.message || result.error || 'Authentication failed');
    }

    const data = result.data as LoginResponse;
    const nextExpiresAt = Date.now() + data.expires_in * 1000;
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt: nextExpiresAt,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async loginWithEmail(credentials: LoginRequest): Promise<LoginResponse> {
    return this.authenticate('/api/v1/auth/login', credentials);
  }

  static async registerWithEmail(credentials: RegisterRequest): Promise<LoginResponse> {
    return this.authenticate('/api/v1/auth/register', credentials);
  }

  static async logout(): Promise<void> {
    if (refreshToken) {
      try {
        await fetch(`${getServerUrl()}/api/v1/auth/logout`, {
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

    // Broadcast the logout so AuthContext resets and redirects to login.
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

  /** True if the access token is expired or within the refresh buffer. */
  static async isTokenExpired(): Promise<boolean> {
    if (!expiresAt) {
      const expiresAtStr = await getStorage().getItem(TOKEN_EXPIRES_AT_KEY);
      expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
    }

    // No expiry stored → assume expired so a refresh is attempted.
    if (!expiresAt) {
      return true;
    }

    return Date.now() >= expiresAt - TOKEN_EXPIRATION_BUFFER_MS;
  }

  static async hasRefreshToken(): Promise<boolean> {
    if (!refreshToken) {
      refreshToken = await getStorage().getItem(REFRESH_TOKEN_KEY);
    }
    return refreshToken !== null;
  }

  /**
   * Ensure a usable access token is available, refreshing proactively if it is
   * expired or expiring soon. Concurrent callers share one refresh promise.
   * Returns true if the caller may proceed, false if the user must re-login.
   *
   * A refresh that fails because the server was unreachable (RefreshNetworkError)
   * returns true: the existing token is kept and the request is allowed through
   * — it may still succeed, or 401 and be retried once connectivity returns.
   * Forcing a full re-login on a transient network blip is the bug this avoids.
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

    const hasRefresh = await this.hasRefreshToken();
    if (!hasRefresh) {
      await this.clearTokens();
      return false;
    }

    try {
      await this.refreshAccessToken();
      return true;
    } catch (error) {
      if (error instanceof RefreshNetworkError) {
        return true;
      }
      console.error(
        '[Auth] Failed to refresh token:',
        error instanceof Error ? error.message : String(error),
      );
      return false;
    }
  }

  static async refreshAccessToken(): Promise<void> {
    // Reuse an in-flight refresh rather than racing a second one.
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
      throw new RefreshRejectedError('No refresh token available');
    }

    let response: Response;
    try {
      response = await fetch(`${getServerUrl()}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch {
      // Never reached the server — keep the session, let the caller retry later.
      throw new RefreshNetworkError('Unable to reach the server to refresh the session.');
    }

    if (!response.ok) {
      // The server actively rejected the refresh token: the session is dead.
      let message = 'Failed to refresh token';
      try {
        const parsed = (await this.parseJsonResponse(response)) as {
          message?: string;
          error?: string;
        };
        message = parsed.message || parsed.error || message;
      } catch {
        // Non-JSON error body — the status is enough to know the session is gone.
      }
      await this.clearTokens();
      throw new RefreshRejectedError(message);
    }

    let result: { data?: LoginResponse };
    try {
      result = (await this.parseJsonResponse(response)) as { data?: LoginResponse };
    } catch {
      // 2xx but unparseable — a server problem, not a dead session. Keep tokens.
      throw new RefreshNetworkError('Invalid response from the refresh endpoint.');
    }

    const data = result.data;
    if (!data || !data.access_token || !data.refresh_token || !data.expires_in) {
      throw new RefreshNetworkError('Invalid refresh response: missing required fields');
    }

    const nextExpiresAt = Date.now() + data.expires_in * 1000;
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt: nextExpiresAt,
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
   * Authenticated fetch with proactive refresh and a single reactive 401 retry.
   * Feature services (read, explore) build on this.
   *
   * The 'Session expired. Please log in again.' and 'Not authenticated' messages
   * are load-bearing: apps/mobile/src/utils/retry.ts classifies retryability by
   * lowercased substring match on error.message, so rephrasing either would make
   * an unrecoverable auth failure retryable.
   */
  static async fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const token = await this.getAccessToken();
    if (!token) {
      throw new Error('Not authenticated');
    }

    // Build headers via the Headers API so any HeadersInit shape from callers
    // merges. We own Authorization; callers may still set their own Content-Type.
    const buildHeaders = (bearer: string): Headers => {
      const headers = new Headers(options.headers);
      if (!headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json');
      }
      headers.set('Authorization', `Bearer ${bearer}`);
      return headers;
    };

    const response = await fetch(url, { ...options, headers: buildHeaders(token) });

    if (response.status !== 401) {
      return response;
    }

    // Reactive fallback: refresh once, then retry the request.
    try {
      await this.refreshAccessToken();
    } catch (error) {
      if (error instanceof RefreshNetworkError) {
        // Couldn't reach the server to refresh — keep tokens and surface a
        // retryable error instead of logging the user out.
        throw error;
      }
      // Server rejected the refresh (doRefreshAccessToken already cleared the
      // tokens for that case; clear again to also cover "no refresh token").
      await this.clearTokens();
      throw new Error('Session expired. Please log in again.');
    }

    const newToken = await this.getAccessToken();
    const retryResponse = await fetch(url, { ...options, headers: buildHeaders(newToken ?? '') });

    // H12: a second 401 after a *successful* refresh means even the fresh token
    // is rejected — the session is broken. Log out rather than handing the 401
    // back to a caller that treats it as an ordinary error and leaves the user
    // stuck in an authenticated-looking but dead UI.
    if (retryResponse.status === 401) {
      await this.clearTokens();
      throw new Error('Session expired. Please log in again.');
    }

    return retryResponse;
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
