import { useNetworkState } from 'expo-network';
import { isNetworkStateOffline } from '../utils/network';

/**
 * Thin wrapper around expo-network's useNetworkState(), narrowed to the
 * single boolean the app needs to decide whether to show the offline banner
 * and let callers fall back to local data.
 */
export function useNetworkStatus(): { isOffline: boolean } {
  const state = useNetworkState();
  return { isOffline: isNetworkStateOffline(state) };
}
