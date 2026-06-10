// Web authentication service. Ported from apps/mobile/src/services/auth.ts with two
// deliberate differences (see web_app_requirements.md §Authentication):
//   1. Persistence backend is localStorage (synchronous) instead of AsyncStorage.
//   2. No device-ID / anonymous login — browsers have no expo-application identifier,
//      so web users authenticate with email/password only.
// Token keys, refresh logic, and the 401-retry interceptor mirror mobile verbatim.
import {
  getServerUrl,
  type LoginResponse,
  type LoginRequest,
  type RegisterRequest,
  type AuthTokens,
  type User,
} from '@cairn/shared';

const ACCESS_TOKEN_KEY = '@cairn:access_token';
const REFRESH_TOKEN_KEY = '@cairn:refresh_token';
const USER_KEY = '@cairn:user';
const TOKEN_EXPIRES_AT_KEY = '@cairn:token_expires_at';

// Buffer time before expiration to trigger proactive refresh (5 minutes).
const TOKEN_EXPIRATION_BUFFER_MS = 5 * 60 * 1000;

type AuthStateListener = (isAuthenticated: boolean) => void;

export class AuthService {
  private static accessToken: string | null = null;
  private static refreshToken: string | null = null;
  private static user: User | null = null;
  private static expiresAt: number | null = null;

  private static isRefreshing: boolean = false;
  private static refreshPromise: Promise<void> | null = null;
  private static listeners: Set<AuthStateListener> = new Set();

  private static async parseJsonResponse(response: Response): Promise<unknown> {
    const text = await response.text();
    try {
      return JSON.parse(text);
    } catch {
      throw new Error('Unable to reach the server. Please try again later.');
    }
  }

  /**
   * Register a listener for auth state changes. Returns an unsubscribe function.
   */
  static onAuthStateChange(listener: AuthStateListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  private static notifyListeners(isAuthenticated: boolean): void {
    this.listeners.forEach((listener) => {
      try {
        listener(isAuthenticated);
      } catch (error) {
        console.error('Error in auth state listener:', error);
      }
    });
  }

  static async initialize(): Promise<void> {
    this.accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
    this.refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
    const userJson = localStorage.getItem(USER_KEY);
    this.user = userJson ? (JSON.parse(userJson) as User) : null;
    const expiresAtStr = localStorage.getItem(TOKEN_EXPIRES_AT_KEY);
    this.expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
  }

  private static async authenticate(path: string, body: object): Promise<LoginResponse> {
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
    const expiresAt = Date.now() + data.expires_in * 1000;
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt,
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
    if (this.refreshToken) {
      try {
        await fetch(`${getServerUrl()}/api/v1/auth/logout`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${this.accessToken}`,
          },
          body: JSON.stringify({ refresh_token: this.refreshToken }),
        });
      } catch (error) {
        console.error('Error during logout:', error);
      }
    }

    await this.clearTokens();
  }

  static async saveTokens(tokens: AuthTokens): Promise<void> {
    this.accessToken = tokens.accessToken;
    this.refreshToken = tokens.refreshToken;
    this.expiresAt = tokens.expiresAt || null;

    localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    if (tokens.expiresAt) {
      localStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString());
    }
  }

  static async clearTokens(): Promise<void> {
    this.accessToken = null;
    this.refreshToken = null;
    this.user = null;
    this.expiresAt = null;

    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(TOKEN_EXPIRES_AT_KEY);

    // Notify listeners that the user is no longer authenticated.
    this.notifyListeners(false);
  }

  static async getAccessToken(): Promise<string | null> {
    if (!this.accessToken) {
      this.accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
    }
    return this.accessToken;
  }

  static async isAuthenticated(): Promise<boolean> {
    const token = await this.getAccessToken();
    return token !== null;
  }

  /**
   * Check if the access token is expired or will expire within the refresh buffer.
   */
  static async isTokenExpired(): Promise<boolean> {
    if (!this.expiresAt) {
      const expiresAtStr = localStorage.getItem(TOKEN_EXPIRES_AT_KEY);
      this.expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
    }

    // If no expiration stored, assume expired to trigger refresh.
    if (!this.expiresAt) {
      return true;
    }

    return Date.now() >= this.expiresAt - TOKEN_EXPIRATION_BUFFER_MS;
  }

  static async hasRefreshToken(): Promise<boolean> {
    if (!this.refreshToken) {
      this.refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
    }
    return this.refreshToken !== null;
  }

  /**
   * Ensure a valid access token is available, refreshing proactively if it is
   * expired or expiring soon. Concurrent callers share a single refresh promise.
   * Returns true if a valid token is available, false if the user must re-login.
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
      console.error('[Auth] Failed to refresh token:', error);
      return false;
    }
  }

  static async refreshAccessToken(): Promise<void> {
    // If a refresh is already in flight, reuse it rather than racing a second one.
    if (this.isRefreshing && this.refreshPromise) {
      return this.refreshPromise;
    }

    this.isRefreshing = true;
    this.refreshPromise = this.doRefreshAccessToken();

    try {
      await this.refreshPromise;
    } finally {
      this.isRefreshing = false;
      this.refreshPromise = null;
    }
  }

  private static async doRefreshAccessToken(): Promise<void> {
    if (!this.refreshToken) {
      this.refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
    }

    if (!this.refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await fetch(`${getServerUrl()}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: this.refreshToken }),
      });

      const result = (await this.parseJsonResponse(response)) as {
        data?: LoginResponse;
        message?: string;
        error?: string;
      };

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to refresh token');
      }

      const data = result.data;
      if (!data || !data.access_token || !data.refresh_token || !data.expires_in) {
        throw new Error('Invalid refresh response: missing required fields');
      }

      const expiresAt = Date.now() + data.expires_in * 1000;
      await this.saveTokens({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        expiresAt,
      });
    } catch (error) {
      // Clear tokens on ANY error (network, parse, HTTP) so the user re-authenticates.
      console.error('[Auth] Refresh failed, clearing tokens:', error);
      await this.clearTokens();
      throw error;
    }
  }

  static async saveUser(user: User): Promise<void> {
    this.user = user;
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  }

  static async getUser(): Promise<User | null> {
    if (!this.user) {
      const userJson = localStorage.getItem(USER_KEY);
      this.user = userJson ? (JSON.parse(userJson) as User) : null;
    }
    return this.user;
  }

  static async getUserId(): Promise<string | null> {
    const user = await this.getUser();
    return user?.id || null;
  }

  /**
   * Authenticated fetch with proactive refresh and a single reactive 401 retry.
   * Feature services (read, explore) build on this — mirrors mobile fetchWithAuth.
   */
  static async fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const accessToken = await this.getAccessToken();
    if (!accessToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${accessToken}`,
        ...options.headers,
      },
    });

    // Reactive fallback: refresh once on 401 and retry the request.
    if (response.status === 401) {
      try {
        await this.refreshAccessToken();
        const newAccessToken = await this.getAccessToken();
        return await fetch(url, {
          ...options,
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${newAccessToken}`,
            ...options.headers,
          },
        });
      } catch {
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
