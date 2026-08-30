// Mobile authentication service. The token-refresh state machine, email login,
// changePassword and fetchWithAuth all live in @cairn/shared (parameterized over
// an injected StorageAdapter — mobile wires AsyncStorage in config/init.ts via
// configureStorage). This file adds only the mobile-specific pieces: device-ID
// login/registration, account upgrade, and the 5xx-retrying fetch wrapper.
import * as Application from 'expo-application';
import { Platform } from 'react-native';
import { AuthService as SharedAuthService, getServerUrl } from '@cairn/shared';
import { MobileAuthRequest, User } from '../types';
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

  static async loginWithDevice() {
    const expo_device_id = await this.getDeviceId();
    return this.authenticate('/api/v1/auth/login/mobile', { expo_device_id } as MobileAuthRequest);
  }

  static async registerWithDevice() {
    const expo_device_id = await this.getDeviceId();
    return this.authenticate('/api/v1/auth/register/mobile', {
      expo_device_id,
    } as MobileAuthRequest);
  }

  static async upgradeAccount(email: string, password: string): Promise<User> {
    const userId = await this.getUserId();
    if (!userId) {
      throw new Error('No user found');
    }

    const response = await this.fetchWithAuth(`${getServerUrl()}/api/v1/user/${userId}/upgrade`, {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });

    const result = (await this.parseJsonResponse(response)) as {
      data?: User;
      message?: string;
      error?: string;
    };

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to upgrade account');
    }

    const user = (result.data ?? (result as unknown as User)) as User;
    await this.saveUser(user);
    return user;
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
}
