import React from 'react';
import { Text } from 'react-native';
import { render, renderHook, screen, waitFor, act } from '@testing-library/react-native';
import { AuthProvider, useAuth } from './AuthContext';
import { AuthService, ArticleStore } from '../services';
import { NetworkError } from '../utils';
import { User } from '../types';

// task_cab7: checkAuthStatus is the offline app-launch path. Before this fix,
// ensureValidToken throwing a NetworkError landed in the generic catch and
// called setUser(null) — logging the user out on screen even though their
// tokens were still on disk. On a NetworkError the persisted session must be
// kept instead.

jest.mock('../services', () => ({
  AuthService: {
    initialize: jest.fn(),
    isAuthenticated: jest.fn(),
    ensureValidToken: jest.fn(),
    getUser: jest.fn(),
    onAuthStateChange: jest.fn(() => () => {}),
    logout: jest.fn(),
  },
  ArticleStore: {
    clear: jest.fn(),
  },
}));

jest.mock('@cairn/shared', () => ({
  loadServerUrl: jest.fn().mockResolvedValue('https://api.test'),
}));

const mockedAuthService = AuthService as jest.Mocked<typeof AuthService>;
const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;

const STORED_USER: User = {
  id: 'user-1',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  lastLoginAt: '2026-01-01T00:00:00Z',
};

function Consumer() {
  const { user, isLoading } = useAuth();
  if (isLoading) return <Text>loading</Text>;
  return <Text>user:{user ? user.id : 'none'}</Text>;
}

describe('AuthProvider.checkAuthStatus offline handling (task_cab7)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'log').mockImplementation(() => {});
    mockedAuthService.onAuthStateChange.mockReturnValue(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('keeps the persisted session when ensureValidToken throws a NetworkError', async () => {
    mockedAuthService.isAuthenticated.mockResolvedValue(true);
    mockedAuthService.ensureValidToken.mockRejectedValue(new NetworkError());
    mockedAuthService.getUser.mockResolvedValue(STORED_USER);

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByText('user:user-1')).toBeTruthy());
    expect(mockedAuthService.getUser).toHaveBeenCalled();
  });

  it('still clears the session when the refresh is genuinely rejected', async () => {
    mockedAuthService.isAuthenticated.mockResolvedValue(true);
    mockedAuthService.ensureValidToken.mockResolvedValue(false);

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByText('user:none')).toBeTruthy());
  });

  it('falls back to a null user, without an unhandled rejection, when getUser throws on corrupt storage', async () => {
    mockedAuthService.isAuthenticated.mockResolvedValue(true);
    mockedAuthService.ensureValidToken.mockRejectedValue(new NetworkError());
    mockedAuthService.getUser.mockRejectedValue(new SyntaxError('Unexpected token in JSON'));

    render(
      <AuthProvider>
        <Consumer />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByText('user:none')).toBeTruthy());
  });
});

describe('AuthProvider.logout', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'log').mockImplementation(() => {});
    mockedAuthService.onAuthStateChange.mockReturnValue(() => {});
    mockedAuthService.isAuthenticated.mockResolvedValue(false);
    mockedAuthService.logout.mockResolvedValue(undefined);
    mockedArticleStore.clear.mockResolvedValue(undefined);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('clears the local article store alongside the auth session', async () => {
    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      await result.current.logout();
    });

    expect(mockedArticleStore.clear).toHaveBeenCalled();
  });

  it('still logs the user out when ArticleStore.clear rejects', async () => {
    mockedArticleStore.clear.mockRejectedValue(new Error('database is locked'));

    const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await expect(
      act(async () => {
        await result.current.logout();
      }),
    ).resolves.toBeUndefined();

    expect(result.current.user).toBeNull();
  });
});
