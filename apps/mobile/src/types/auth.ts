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
}

export interface LoginResponse {
  user: User;
  tokens: AuthTokens;
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
  deviceId: string;
}
