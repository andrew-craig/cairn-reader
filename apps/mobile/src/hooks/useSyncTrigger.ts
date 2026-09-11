import { useEffect, useRef } from 'react';
import { AppState, AppStateStatus } from 'react-native';
import { useNetworkStatus } from './useNetworkStatus';
import { SyncTrigger } from '../services/syncTrigger';

/**
 * Fires SyncTrigger.run() on a *transition* to the foreground and on a
 * *transition* from offline to online — never merely because the app is
 * currently active or currently online. Mount once inside the authenticated
 * tree (see SyncTriggerEffect / RootNavigator, next to OfflineBanner) so it
 * never runs logged out.
 */
export function useSyncTrigger(): void {
  const { isOffline } = useNetworkStatus();
  const wasOfflineRef = useRef(isOffline);

  useEffect(() => {
    if (wasOfflineRef.current && !isOffline) {
      void SyncTrigger.run();
    }
    wasOfflineRef.current = isOffline;
  }, [isOffline]);

  useEffect(() => {
    // The tree this hook mounts in only renders while the app is in the
    // foreground, so 'active' is the correct starting assumption — this
    // only needs to catch *later* transitions, not classify the state at
    // mount time.
    let appState: AppStateStatus = 'active';
    const subscription = AppState.addEventListener('change', (nextState: AppStateStatus) => {
      const prevState = appState;
      appState = nextState;
      if (nextState === 'active' && prevState !== 'active') {
        void SyncTrigger.run();
      }
    });
    return () => subscription.remove();
  }, []);
}
