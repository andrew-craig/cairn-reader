import React from 'react';
import { Alert } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import { LoginScreen } from './LoginScreen';
import { AuthService } from '../services';
import { useNetworkStatus } from '../hooks/useNetworkStatus';
import { NetworkError } from '../utils/errors';
import { LoginResponse } from '../types';

// task_5bd6: offline-aware login. handleGetStarted's bare `catch {}` used to
// swallow a NetworkError from loginWithDevice and fall through to
// registerWithDevice — a second doomed round trip while offline. These tests
// fail against pre-fix code: the offline case previously called
// registerWithDevice once (it should never be called), and the offline-copy
// tests fail because LoginScreen never read network status at all.

jest.mock('../services', () => ({
  AuthService: {
    loginWithDevice: jest.fn(),
    registerWithDevice: jest.fn(),
    loginWithEmail: jest.fn(),
    clearTokens: jest.fn(),
  },
}));

jest.mock('../hooks/useNetworkStatus');

jest.mock('@cairn/shared', () => ({
  getServerUrl: jest.fn(() => 'https://api.test'),
  setServerUrl: jest.fn(),
}));

const mockedAuthService = AuthService as jest.Mocked<typeof AuthService>;
const mockedUseNetworkStatus = useNetworkStatus as jest.Mock;

const LOGIN_RESPONSE: LoginResponse = {
  user: {
    id: 'user-1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    lastLoginAt: '2026-01-01T00:00:00Z',
  },
  access_token: 'access',
  refresh_token: 'refresh',
  expires_in: 3600,
};

describe('LoginScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('makes exactly one network attempt when device login fails offline, and does not fall back to register', async () => {
    mockedAuthService.loginWithDevice.mockRejectedValue(new NetworkError());
    const onLoginSuccess = jest.fn();

    render(<LoginScreen onLoginSuccess={onLoginSuccess} />);
    fireEvent.press(screen.getByText('Explore'));

    await waitFor(() => expect(Alert.alert).toHaveBeenCalled());

    expect(mockedAuthService.loginWithDevice).toHaveBeenCalledTimes(1);
    expect(mockedAuthService.registerWithDevice).not.toHaveBeenCalled();
    expect(onLoginSuccess).not.toHaveBeenCalled();
    expect(Alert.alert).toHaveBeenCalledWith('Error', expect.stringMatching(/reach the server/i));
  });

  it('still falls back to register when device login is rejected for a real (non-network) reason', async () => {
    mockedAuthService.loginWithDevice.mockRejectedValue(new Error('Device login failed'));
    mockedAuthService.registerWithDevice.mockResolvedValue(LOGIN_RESPONSE);
    const onLoginSuccess = jest.fn();

    render(<LoginScreen onLoginSuccess={onLoginSuccess} />);
    fireEvent.press(screen.getByText('Explore'));

    await waitFor(() => expect(onLoginSuccess).toHaveBeenCalled());
    expect(mockedAuthService.loginWithDevice).toHaveBeenCalledTimes(1);
    expect(mockedAuthService.registerWithDevice).toHaveBeenCalledTimes(1);
  });

  it('shows offline-specific copy when offline', () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: true });
    render(<LoginScreen onLoginSuccess={jest.fn()} />);

    expect(screen.getByText(/signing in needs a connection/i)).toBeTruthy();
  });

  it('does not show offline copy when online', () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    render(<LoginScreen onLoginSuccess={jest.fn()} />);

    expect(screen.queryByText(/signing in needs a connection/i)).toBeNull();
  });

  it('keeps the submit button pressable while offline (not disabled)', async () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: true });
    mockedAuthService.loginWithDevice.mockResolvedValue(LOGIN_RESPONSE);
    const onLoginSuccess = jest.fn();

    render(<LoginScreen onLoginSuccess={onLoginSuccess} />);
    fireEvent.press(screen.getByText('Explore'));

    await waitFor(() => expect(onLoginSuccess).toHaveBeenCalled());
  });
});
