// Mobile's authentication service. The token-refresh state machine, email
// login/register, logout, changePassword and fetchWithAuth all live in
// @cairn/shared (task_47c1) — this subclass adds only what is genuinely
// mobile-specific: the device-ID login flavors, account upgrade, and the
// retrying fetch wrapper built on mobile's withRetry.
//
// Session state is module-level inside the shared module, so it stays a single
// session no matter whether a caller reaches it through this subclass or the
// base class.
import * as Application from 'expo-application';
import { Platform } from 'react-native';
import {
  AuthService as SharedAuthService,
  getServerUrl,
  fetchOrNetworkError,
  type LoginResponse,
  type MobileAuthRequest,
  type User,
} from '@cairn/shared';
import { withRetry } from '../utils/retry';

export class AuthService extends SharedAuthService {
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
    return this.authenticate(
      '/api/v1/auth/login/mobile',
      { expo_device_id: deviceId } as MobileAuthRequest,
      'Device login failed',
    );
  }

  static async registerWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();
    return this.authenticate(
      '/api/v1/auth/register/mobile',
      { expo_device_id: deviceId } as MobileAuthRequest,
      'Device registration failed',
    );
  }

  /**
   * Like fetchWithAuth but also retries on 5xx / network errors.
   * 4xx responses are returned as-is (callers handle error status).
   */
  static async fetchWithAuthAndRetry(url: string, options: RequestInit = {}): Promise<Response> {
    return withRetry(async (signal) => {
      const response = await this.fetchWithAuth(url, { ...options, signal });
      // Throw on 5xx so withRetry can retry; let 4xx pass through to caller
      if (response.status >= 500) {
        throw new Error(`Server error ${response.status}`);
      }
      return response;
    });
  }

  /** Turn an anonymous device account into an email/password account. */
  static async upgradeAccount(email: string, password: string): Promise<User> {
    const isValid = await this.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const userId = await this.getUserId();
    if (!userId) {
      throw new Error('No user found');
    }

    const accessToken = await this.getAccessToken();
    const response = await fetchOrNetworkError(`${getServerUrl()}/api/v1/user/${userId}/upgrade`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${accessToken}`,
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
}
