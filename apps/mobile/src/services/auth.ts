import AsyncStorage from '@react-native-async-storage/async-storage';
import * as Application from 'expo-application';
import {
  LoginResponse,
  LoginRequest,
  RegisterRequest,
  MobileAuthRequest,
  AuthTokens,
} from '../types';

const API_BASE_URL = 'https://cairn.seatrain.net';
const ACCESS_TOKEN_KEY = '@cairn:access_token';
const REFRESH_TOKEN_KEY = '@cairn:refresh_token';

export class AuthService {
  private static accessToken: string | null = null;
  private static refreshToken: string | null = null;

  static async initialize(): Promise<void> {
    this.accessToken = await AsyncStorage.getItem(ACCESS_TOKEN_KEY);
    this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
  }

  static async getDeviceId(): Promise<string> {
    const deviceId = await Application.getInstallationIdAsync();
    if (!deviceId) {
      throw new Error('Failed to get device ID');
    }
    return deviceId;
  }

  static async loginWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();

    const response = await fetch(`${API_BASE_URL}/auth/login/mobile`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ deviceId } as MobileAuthRequest),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Device login failed: ${error}`);
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens(data.tokens);
    return data;
  }

  static async registerWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();

    const response = await fetch(`${API_BASE_URL}/auth/register/mobile`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ deviceId } as MobileAuthRequest),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Device registration failed: ${error}`);
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens(data.tokens);
    return data;
  }

  static async loginWithEmail(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await fetch(`${API_BASE_URL}/auth/login`, {
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
    await this.saveTokens(data.tokens);
    return data;
  }

  static async registerWithEmail(credentials: RegisterRequest): Promise<LoginResponse> {
    const response = await fetch(`${API_BASE_URL}/auth/register`, {
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
    await this.saveTokens(data.tokens);
    return data;
  }

  static async logout(): Promise<void> {
    if (this.refreshToken) {
      try {
        await fetch(`${API_BASE_URL}/auth/logout`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${this.accessToken}`,
          },
          body: JSON.stringify({ refreshToken: this.refreshToken }),
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

    await AsyncStorage.removeItem(ACCESS_TOKEN_KEY);
    await AsyncStorage.removeItem(REFRESH_TOKEN_KEY);
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

    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refreshToken: this.refreshToken }),
    });

    if (!response.ok) {
      await this.clearTokens();
      throw new Error('Failed to refresh token');
    }

    const data = await response.json();
    await this.saveTokens(data.tokens);
  }
}
