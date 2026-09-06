import { isOffline } from './network';
import { getNetworkStateAsync } from 'expo-network';

jest.mock('expo-network');

describe('isOffline', () => {
  afterEach(() => jest.clearAllMocks());

  it('returns false when the device is connected', async () => {
    (getNetworkStateAsync as jest.Mock).mockResolvedValue({ isConnected: true });
    expect(await isOffline()).toBe(false);
  });

  it('returns true when the device is disconnected', async () => {
    (getNetworkStateAsync as jest.Mock).mockResolvedValue({ isConnected: false });
    expect(await isOffline()).toBe(true);
  });

  it('treats an unknown/undefined connectivity signal as online', async () => {
    (getNetworkStateAsync as jest.Mock).mockResolvedValue({ isConnected: undefined });
    expect(await isOffline()).toBe(false);
  });
});
