import type { StorageAdapter } from '../config/api';

// localStorage-backed implementation of the shared StorageAdapter. The shared
// config/service layer is async (mirroring AsyncStorage on mobile); localStorage
// is synchronous, so the calls are simply wrapped in resolved promises.
// (decision_9b2d ADR: web persists via localStorage.)
export const localStorageAdapter: StorageAdapter = {
  async getItem(key: string): Promise<string | null> {
    return localStorage.getItem(key);
  },
  async setItem(key: string, value: string): Promise<void> {
    localStorage.setItem(key, value);
  },
  async removeItem(key: string): Promise<void> {
    localStorage.removeItem(key);
  },
};
