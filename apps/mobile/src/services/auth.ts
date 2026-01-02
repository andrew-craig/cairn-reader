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
import { API_CONFIG } from '../config/api';

const API_BASE_URL = API_CONFIG.USER_SERVICE_URL;
const ACCESS_TOKEN_KEY = '@cairn:access_token';
const REFRESH_TOKEN_KEY = '@cairn:refresh_token';
const USER_KEY = '@cairn:user';

export class AuthService {
  private static accessToken: string | null = null;
  private static refreshToken: string | null = null;
  private static user: User | null = null;

  static async initialize(): Promise<void> {
    this.accessToken = await AsyncStorage.getItem(ACCESS_TOKEN_KEY);
    this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
    const userJson = await AsyncStorage.getItem(USER_KEY);
    this.user = userJson ? JSON.parse(userJson) : null;
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

    const response = await fetch(`${API_BASE_URL}/api/v1/auth/login/mobile`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ expo_device_id: deviceId } as MobileAuthRequest),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Device login failed: ${error}`);
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async registerWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();

    const response = await fetch(`${API_BASE_URL}/api/v1/auth/register/mobile`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ expo_device_id: deviceId } as MobileAuthRequest),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Device registration failed: ${error}`);
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async loginWithEmail(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Email login failed: ${error}`);
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async registerWithEmail(credentials: RegisterRequest): Promise<LoginResponse> {
    const response = await fetch(`${API_BASE_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Email registration failed: ${error}`);
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
    await this.saveUser(data.user);
    return data;
  }

  static async logout(): Promise<void> {
    if (this.refreshToken) {
      try {
        await fetch(`${API_BASE_URL}/api/v1/auth/logout`, {
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

    await AsyncStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    await AsyncStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
  }

  static async clearTokens(): Promise<void> {
    this.accessToken = null;
    this.refreshToken = null;
    this.user = null;

    await AsyncStorage.removeItem(ACCESS_TOKEN_KEY);
    await AsyncStorage.removeItem(REFRESH_TOKEN_KEY);
    await AsyncStorage.removeItem(USER_KEY);
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

  static async refreshAccessToken(): Promise<void> {
    if (!this.refreshToken) {
      this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
    }

    if (!this.refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refresh_token: this.refreshToken }),
    });

    if (!response.ok) {
      await this.clearTokens();
      throw new Error('Failed to refresh token');
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
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
}
