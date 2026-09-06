import { renderHook } from '@testing-library/react-native';
import { useNetworkStatus } from './useNetworkStatus';
import { useNetworkState } from 'expo-network';

jest.mock('expo-network');

describe('useNetworkStatus', () => {
  afterEach(() => jest.clearAllMocks());

  it('reports offline when isConnected is false', () => {
    (useNetworkState as jest.Mock).mockReturnValue({ isConnected: false });
    const { result } = renderHook(() => useNetworkStatus());
    expect(result.current.isOffline).toBe(true);
  });

  it('reports online when isConnected is true', () => {
    (useNetworkState as jest.Mock).mockReturnValue({ isConnected: true });
    const { result } = renderHook(() => useNetworkStatus());
    expect(result.current.isOffline).toBe(false);
  });

  it('treats an unknown/undefined connectivity signal as online', () => {
    (useNetworkState as jest.Mock).mockReturnValue({ isConnected: undefined });
    const { result } = renderHook(() => useNetworkStatus());
    expect(result.current.isOffline).toBe(false);
  });
});
