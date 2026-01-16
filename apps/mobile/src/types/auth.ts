export interface User {
  id: string;
  email?: string;
  expoDeviceId?: string;
  createdAt: string;
  updatedAt: string;
  lastLoginAt: string;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresAt?: number; // Unix timestamp in milliseconds
}

export interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface MobileAuthRequest {
  expo_device_id: string;
}
