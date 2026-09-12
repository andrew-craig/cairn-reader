import React from 'react';
import { Alert } from 'react-native';
import { render, screen, fireEvent, waitFor } from '@testing-library/react-native';
import { LoginScreen } from './LoginScreen';
import { AuthService } from '../services';
import { useNetworkStatus } from '../hooks/useNetworkStatus';
import { NetworkError, HttpError } from '../utils/errors';
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

jest.mock('expo-application', () => ({
  getIosIdForVendorAsync: jest.fn().mockResolvedValue('test-device-id'),
  getAndroidId: jest.fn().mockReturnValue('test-device-id'),
}));

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
  const originalFetch = global.fetch;

  beforeEach(() => {
    jest.clearAllMocks();
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
    // jest.restoreAllMocks() only undoes jest.spyOn spies — it doesn't touch
    // a direct `global.fetch = ...` assignment (see the unparseable-body
    // test below), so restore it explicitly or the stub leaks into later
    // tests in this file.
    global.fetch = originalFetch;
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

  it('does not fall back to register when device login fails on an unparseable response body (e.g. captive portal)', async () => {
    // Exercises the real AuthService.loginWithDevice (not the mock above) so
    // this test proves out auth.ts's parseJsonResponse, not just LoginScreen's
    // own instanceof check. A captive-portal WiFi network responds 200 with an
    // HTML login page instead of JSON; that must surface as NetworkError, or
    // this screen's login-then-register fallback runs anyway and the user
    // waits through a second doomed round trip before seeing the error.
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { AuthService: RealAuthService } = jest.requireActual('../services');
    global.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve('<html><body>Please log in to the WiFi network</body></html>'),
    }) as unknown as typeof fetch;
    mockedAuthService.loginWithDevice.mockImplementationOnce(() => RealAuthService.loginWithDevice());
    const onLoginSuccess = jest.fn();

    render(<LoginScreen onLoginSuccess={onLoginSuccess} />);
    fireEvent.press(screen.getByText('Explore'));

    await waitFor(() => expect(Alert.alert).toHaveBeenCalled());

    expect(mockedAuthService.loginWithDevice).toHaveBeenCalledTimes(1);
    expect(mockedAuthService.registerWithDevice).not.toHaveBeenCalled();
    expect(onLoginSuccess).not.toHaveBeenCalled();
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

  // task_f19d: a 5xx device login is a broken server, not a rejected
  // credential, and must not trigger the register fallback — the same
  // symptom task_5bd6 fixed for NetworkError, still reachable via HttpError
  // before this fix (pre-change, only `instanceof NetworkError` was checked,
  // so a 500 fell through to registerWithDevice() same as a 4xx).
  it('makes exactly one network attempt and does not fall back to register when device login fails with a 5xx', async () => {
    mockedAuthService.loginWithDevice.mockRejectedValue(new HttpError(500, 'Internal server error'));
    const onLoginSuccess = jest.fn();

    render(<LoginScreen onLoginSuccess={onLoginSuccess} />);
    fireEvent.press(screen.getByText('Explore'));

    await waitFor(() => expect(Alert.alert).toHaveBeenCalled());

    expect(mockedAuthService.loginWithDevice).toHaveBeenCalledTimes(1);
    expect(mockedAuthService.registerWithDevice).not.toHaveBeenCalled();
    expect(onLoginSuccess).not.toHaveBeenCalled();
    expect(Alert.alert).toHaveBeenCalledWith('Error', 'Internal server error');
  });

  // Regression guard: a 4xx device login is a definitive rejection and must
  // still fall back to register — the only case the fallback was ever meant
  // for. Passes both before and after this change.
  it('still falls back to register when device login fails with a 4xx', async () => {
    mockedAuthService.loginWithDevice.mockRejectedValue(new HttpError(401, 'Invalid device credential'));
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
