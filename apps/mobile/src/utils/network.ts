import { getNetworkStateAsync, NetworkState } from 'expo-network';

/**
 * An unknown/undefined connectivity signal (e.g. the OS hasn't determined
 * network state yet) is treated as online — never block the UI on a signal
 * that isn't reliable. Only an explicit `isConnected: false` counts as
 * offline.
 */
export function isNetworkStateOffline(state: NetworkState): boolean {
  return state.isConnected === false;
}

/**
 * One-off connectivity check for callers that cannot use a hook (services,
 * plain utility functions). Components should use useNetworkStatus instead.
 */
export async function isOffline(): Promise<boolean> {
  const state = await getNetworkStateAsync();
  return isNetworkStateOffline(state);
}
