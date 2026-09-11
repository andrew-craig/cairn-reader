import { renderHook } from '@testing-library/react-native';
import { AppState } from 'react-native';
import { useSyncTrigger } from './useSyncTrigger';
import { useNetworkStatus } from './useNetworkStatus';
import { SyncTrigger } from '../services/syncTrigger';

// task_06e5: fires on an offline->online transition and on an
// AppState background/inactive->active transition — never merely because
// the app is currently online/active, and never twice for one transition.

jest.mock('./useNetworkStatus');
jest.mock('../services/syncTrigger', () => ({
  SyncTrigger: { run: jest.fn() },
}));

const mockedUseNetworkStatus = useNetworkStatus as jest.Mock;
const mockedSyncTrigger = SyncTrigger as jest.Mocked<typeof SyncTrigger>;

// Grabs the listener the hook registered with AppState.addEventListener so
// the test can simulate a state change directly, the way the default
// react-native jest mock (react-native/jest/mocks/AppState.js) is designed
// to be driven.
const emitAppStateChange = (nextState: string) => {
  const addEventListener = AppState.addEventListener as jest.Mock;
  const call = addEventListener.mock.calls[addEventListener.mock.calls.length - 1];
  const listener = call[1] as (state: string) => void;
  listener(nextState);
};

describe('useSyncTrigger', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
  });

  it('fires on an offline->online transition', () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: true });
    const { rerender } = renderHook(() => useSyncTrigger());
    expect(mockedSyncTrigger.run).not.toHaveBeenCalled();

    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    rerender(undefined);

    expect(mockedSyncTrigger.run).toHaveBeenCalledTimes(1);
  });

  it('does not fire on a re-render with unchanged offline state', () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    const { rerender } = renderHook(() => useSyncTrigger());

    rerender(undefined);
    rerender(undefined);

    expect(mockedSyncTrigger.run).not.toHaveBeenCalled();
  });

  it('does not fire merely from being online at mount (no prior offline state)', () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    renderHook(() => useSyncTrigger());

    expect(mockedSyncTrigger.run).not.toHaveBeenCalled();
  });

  it('fires on an AppState background->active transition', () => {
    renderHook(() => useSyncTrigger());

    emitAppStateChange('background');
    expect(mockedSyncTrigger.run).not.toHaveBeenCalled();

    emitAppStateChange('active');
    expect(mockedSyncTrigger.run).toHaveBeenCalledTimes(1);
  });

  it('does not fire on active->active', () => {
    renderHook(() => useSyncTrigger());

    emitAppStateChange('active');

    expect(mockedSyncTrigger.run).not.toHaveBeenCalled();
  });

  it('does not fire twice for the same background->active transition', () => {
    renderHook(() => useSyncTrigger());

    emitAppStateChange('inactive');
    emitAppStateChange('active');
    emitAppStateChange('active');

    expect(mockedSyncTrigger.run).toHaveBeenCalledTimes(1);
  });
});
