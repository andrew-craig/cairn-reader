import AsyncStorage from '@react-native-async-storage/async-storage';
import * as Application from 'expo-application';
import { Platform } from 'react-native';
import {
  LoginResponse,
  LoginRequest,
  RegisterRequest,
  MobileAuthRequest,
  AuthTokens,
  User,
} from '../types';
import { getServerUrl } from '@cairn/shared';
import { withRetry } from '../utils/retry';
import { NetworkError } from '../utils/errors';

const ACCESS_TOKEN_KEY = '@cairn:access_token';
const REFRESH_TOKEN_KEY = '@cairn:refresh_token';
const USER_KEY = '@cairn:user';
const TOKEN_EXPIRES_AT_KEY = '@cairn:token_expires_at';

// Buffer time before expiration to trigger proactive refresh (5 minutes)
const TOKEN_EXPIRATION_BUFFER_MS = 5 * 60 * 1000;

// Listener type for auth state changes
type AuthStateListener = (isAuthenticated: boolean) => void;

export class AuthService {
  private static accessToken: string | null = null;
  private static refreshToken: string | null = null;
  private static user: User | null = null;
  private static expiresAt: number | null = null;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private static async parseJsonResponse(response: Response): Promise<any> {
    const text = await response.text();
    try {
      return JSON.parse(text);
    } catch {
      throw new Error('Unable to reach the server. Please try again later.');
    }
  }

  private static isRefreshing: boolean = false;
  private static refreshPromise: Promise<void> | null = null;
  private static listeners: Set<AuthStateListener> = new Set();

  /**
   * Register a listener for auth state changes.
   * Returns an unsubscribe function.
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
    this.accessToken = await AsyncStorage.getItem(ACCESS_TOKEN_KEY);
    this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
    const userJson = await AsyncStorage.getItem(USER_KEY);
    this.user = userJson ? JSON.parse(userJson) : null;
    const expiresAtStr = await AsyncStorage.getItem(TOKEN_EXPIRES_AT_KEY);
    this.expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
  }

  static async getDeviceId(): Promise<string> {
    let deviceId: string | null = null;

    if (Platform.OS === 'ios') {
      deviceId = await Application.getIosIdForVendorAsync();
    } else if (Platform.OS === 'android') {
      deviceId = Application.getAndroidId();
    }

    if (!deviceId) {
      throw new Error('Failed to get device ID');
    }
    return deviceId;
  }

  static async loginWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();

    const response = await fetch(`${getServerUrl()}/api/v1/auth/login/mobile`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ expo_device_id: deviceId } as MobileAuthRequest),
    });

    const result = await this.parseJsonResponse(response);

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Device login failed');
    }

    const data: LoginResponse = result.data;
    const expiresAt = Date.now() + (data.expires_in * 1000);
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async registerWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();

    const response = await fetch(`${getServerUrl()}/api/v1/auth/register/mobile`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ expo_device_id: deviceId } as MobileAuthRequest),
    });

    const result = await this.parseJsonResponse(response);

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Device registration failed');
    }

    const data: LoginResponse = result.data;
    const expiresAt = Date.now() + (data.expires_in * 1000);
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async loginWithEmail(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await fetch(`${getServerUrl()}/api/v1/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    });

    const result = await this.parseJsonResponse(response);

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Email login failed');
    }

    const data: LoginResponse = result.data;
    const expiresAt = Date.now() + (data.expires_in * 1000);
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async registerWithEmail(credentials: RegisterRequest): Promise<LoginResponse> {
    const response = await fetch(`${getServerUrl()}/api/v1/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    });

    const result = await this.parseJsonResponse(response);

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Email registration failed');
    }

    const data: LoginResponse = result.data;
    const expiresAt = Date.now() + (data.expires_in * 1000);
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async logout(): Promise<void> {
    if (this.refreshToken) {
      try {
        await fetch(`${getServerUrl()}/api/v1/auth/logout`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${this.accessToken}`,
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

    await AsyncStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    await AsyncStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
    if (tokens.expiresAt) {
      await AsyncStorage.setItem(TOKEN_EXPIRES_AT_KEY, tokens.expiresAt.toString());
    }
  }

  static async clearTokens(): Promise<void> {
    this.accessToken = null;
    this.refreshToken = null;
    this.user = null;
    this.expiresAt = null;

    await AsyncStorage.removeItem(ACCESS_TOKEN_KEY);
    await AsyncStorage.removeItem(REFRESH_TOKEN_KEY);
    await AsyncStorage.removeItem(USER_KEY);
    await AsyncStorage.removeItem(TOKEN_EXPIRES_AT_KEY);

    // Notify listeners that user is no longer authenticated
    this.notifyListeners(false);
  }

  static async getAccessToken(): Promise<string | null> {
    if (!this.accessToken) {
      this.accessToken = await AsyncStorage.getItem(ACCESS_TOKEN_KEY);
    }
    return this.accessToken;
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
    if (!this.expiresAt) {
      const expiresAtStr = await AsyncStorage.getItem(TOKEN_EXPIRES_AT_KEY);
      this.expiresAt = expiresAtStr ? parseInt(expiresAtStr, 10) : null;
    }

    // If no expiration stored, assume expired to trigger refresh
    if (!this.expiresAt) {
      return true;
    }

    // Consider expired if within buffer time of expiration
    return Date.now() >= (this.expiresAt - TOKEN_EXPIRATION_BUFFER_MS);
  }

  /**
   * Check if we have a refresh token available.
   */
  static async hasRefreshToken(): Promise<boolean> {
    if (!this.refreshToken) {
      this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
    }
    return this.refreshToken !== null;
  }

  /**
   * Ensure we have a valid access token, refreshing if necessary.
   * This method handles concurrent refresh requests by reusing the same promise.
   * Returns true if a valid token is available, false if user needs to re-login.
   */
  static async ensureValidToken(): Promise<boolean> {
    const hasToken = await this.isAuthenticated();
    if (!hasToken) {
      if (__DEV__) {
        console.log('[Auth] No access token found, authentication required');
      }
      return false;
    }

    const isExpired = await this.isTokenExpired();
    if (!isExpired) {
      if (__DEV__) {
        const timeLeft = this.expiresAt ? Math.floor((this.expiresAt - Date.now()) / 1000) : 'unknown';
        console.log(`[Auth] Access token still valid (${timeLeft}s remaining)`);
      }
      return true;
    }

    if (__DEV__) {
      console.log('[Auth] Access token expired or expiring soon, attempting refresh');
    }

    // Token is expired, need to refresh
    const hasRefresh = await this.hasRefreshToken();
    if (!hasRefresh) {
      if (__DEV__) {
        console.log('[Auth] No refresh token available, clearing tokens');
      }
      await this.clearTokens();
      return false;
    }

    try {
      await this.refreshAccessToken();
      if (__DEV__) {
        console.log('[Auth] Token refresh successful');
      }
      return true;
    } catch (error) {
      if (error instanceof NetworkError) {
        // Server unreachable — not a rejection. Let the caller distinguish
        // this from a genuine "please log in again" by propagating it.
        throw error;
      }
      console.error('[Auth] Failed to refresh token:', error instanceof Error ? error.message : String(error));
      return false;
    }
  }

  static async refreshAccessToken(): Promise<void> {
    // If already refreshing, wait for the existing refresh to complete
    if (this.isRefreshing && this.refreshPromise) {
      if (__DEV__) {
        console.log('[Auth] Refresh already in progress, waiting for existing refresh');
      }
      return this.refreshPromise;
    }

    if (__DEV__) {
      console.log('[Auth] Starting token refresh');
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
      this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
    }

    if (!this.refreshToken) {
      console.error('[Auth] No refresh token available in doRefreshAccessToken');
      throw new Error('No refresh token available');
    }

    if (__DEV__) {
      console.log('[Auth] Attempting token refresh');
    }

    let response: Response;
    try {
      const requestBody = { refresh_token: this.refreshToken };

      response = await fetch(`${getServerUrl()}/api/v1/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });
    } catch (error) {
      // Could not reach the server at all (offline, timeout, DNS, etc.). This is
      // not a rejection of the credential, so the refresh token must survive.
      console.error('[Auth] Refresh request failed, server unreachable:', error instanceof Error ? error.message : String(error));
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
      if (__DEV__) {
        console.error('[Auth] Refresh failed with error response:', {
          status: response.status,
          message: result.message || result.error,
        });
      } else {
        console.error('[Auth] Refresh failed with error response, status:', response.status);
      }

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
      console.error('[Auth] Refresh response missing required fields:', {
        hasData: !!data,
        hasAccessToken: !!data?.access_token,
        hasRefreshToken: !!data?.refresh_token,
        hasExpiresIn: !!data?.expires_in,
      });
      throw new NetworkError();
    }

    const expiresAt = Date.now() + (data.expires_in * 1000);

    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiresAt,
    });

    if (__DEV__) {
      console.log('[Auth] Token refresh successful');
    }
  }

  static async saveUser(user: User): Promise<void> {
    this.user = user;
    await AsyncStorage.setItem(USER_KEY, JSON.stringify(user));
  }

  static async getUser(): Promise<User | null> {
    if (!this.user) {
      const userJson = await AsyncStorage.getItem(USER_KEY);
      this.user = userJson ? JSON.parse(userJson) : null;
    }
    return this.user;
  }

  static async getUserId(): Promise<string | null> {
    const user = await this.getUser();
    return user?.id || null;
  }

  /**
   * fetch(), converting "couldn't reach the server" into a NetworkError so
   * callers can tell it apart from a rejected credential. Anything else is
   * rethrown untouched. Used by both the primary request and the 401 retry
   * in fetchWithAuth — a dropped connection in either window must not be
   * mistaken for the server rejecting the credential.
   */
  private static async fetchOrNetworkError(url: string, init: RequestInit): Promise<Response> {
    try {
      return await fetch(url, init);
    } catch (error) {
      // An aborted fetch can reject with a DOMException, which does not extend Error
      // in every runtime, so match on the name rather than the type.
      if (error instanceof TypeError || (error as { name?: string } | null)?.name === 'AbortError') {
        throw new NetworkError();
      }
      throw error;
    }
  }

  /**
   * Authenticated fetch with proactive token refresh and a single reactive 401
   * retry. Feature services (read, explore) build on this — one implementation,
   * not a private copy per service.
   *
   * The 'Session expired. Please log in again.' and 'Not authenticated' error
   * messages are load-bearing: utils/retry.ts classifies retryability by
   * substring-matching the message, so rephrasing either makes an unrecoverable
   * auth failure retryable.
   */
  static async fetchWithAuth(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
    // Proactively check and refresh token if expired before making request
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const accessToken = await this.getAccessToken();

    if (!accessToken) {
      throw new Error('Not authenticated');
    }

    const response = await this.fetchOrNetworkError(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
        ...options.headers,
      },
    });

    // Handle 401 Unauthorized - try to refresh token (fallback for edge cases)
    if (response.status === 401) {
      try {
        await this.refreshAccessToken();
        const newAccessToken = await this.getAccessToken();

        // Retry the request with new token
        const retryResponse = await this.fetchOrNetworkError(url, {
          ...options,
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${newAccessToken}`,
            ...options.headers,
          },
        });

        return retryResponse;
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

  /**
   * Like fetchWithAuth but also retries on 5xx / network errors.
   * 4xx responses are returned as-is (callers handle error status).
   */
  static async fetchWithAuthAndRetry(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
    return withRetry(async (signal) => {
      const response = await this.fetchWithAuth(url, { ...options, signal });
      // Throw on 5xx so withRetry can retry; let 4xx pass through to caller
      if (response.status >= 500) {
        throw new Error(`Server error ${response.status}`);
      }
      return response;
    });
  }

  static async upgradeAccount(email: string, password: string): Promise<User> {
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const userId = await this.getUserId();
    if (!userId) {
      throw new Error('No user found');
    }

    const response = await fetch(`${getServerUrl()}/api/v1/user/${userId}/upgrade`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`,
      },
      body: JSON.stringify({ email, password }),
    });

    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to upgrade account');
    }

    const user: User = result.data || result;
    await this.saveUser(user);
    return user;
  }

  static async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const userId = await this.getUserId();
    if (!userId) {
      throw new Error('No user found');
    }

    const response = await fetch(`${getServerUrl()}/api/v1/user/${userId}/password`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.accessToken}`,
      },
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    });

    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to change password');
    }
  }
}
