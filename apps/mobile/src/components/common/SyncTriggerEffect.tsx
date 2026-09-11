import { useSyncTrigger } from '../../hooks/useSyncTrigger';

/**
 * Renders nothing — mounts useSyncTrigger's AppState/connectivity listener.
 * A hook can't be called conditionally inside RootNavigator itself (it
 * returns early before the authenticated tree for the loading/logged-out
 * states), so — like OfflineBanner — this is a separate component rendered
 * only inside that tree, which is what keeps the trigger from ever running
 * logged out.
 */
export function SyncTriggerEffect(): null {
  useSyncTrigger();
  return null;
}
