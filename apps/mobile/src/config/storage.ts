import AsyncStorage from '@react-native-async-storage/async-storage';
import type { StorageAdapter } from '@cairn/shared';

// Mobile-specific glue for the shared API config layer (decision_9b2d ADR). The
// server-URL state machine lives in @cairn/shared, parameterized over a default
// URL and a StorageAdapter; mobile supplies both. App.tsx wires these in at
// startup via configureStorage() and configureDefaultServerUrl().

// The backend the mobile app ships against by default. Unlike web (which serves
// the API from its own origin), the mobile build talks to a fixed backend.
export const DEFAULT_SERVER_URL = 'https://cairn.seatrain.net';

// AsyncStorage-backed implementation of the shared StorageAdapter.
export const asyncStorageAdapter: StorageAdapter = {
  getItem: (key) => AsyncStorage.getItem(key),
  setItem: (key, value) => AsyncStorage.setItem(key, value),
  removeItem: (key) => AsyncStorage.removeItem(key),
};
