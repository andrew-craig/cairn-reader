// Server URL configuration, ported from apps/mobile/src/config/api.ts.
//
// The mobile version imports AsyncStorage directly; here the persistence
// backend is injected via a StorageAdapter so each client supplies its own
// (AsyncStorage on mobile, localStorage on web — see decision_9b2d ADR).

export interface StorageAdapter {
  getItem(key: string): Promise<string | null>;
  setItem(key: string, value: string): Promise<void>;
  removeItem(key: string): Promise<void>;
}

export const DEFAULT_SERVER_URL = 'https://cairn.seatrain.net';
const SERVER_URL_KEY = '@cairn:server_url';

// Persisted session keys. Tokens are issued by (and valid for) a specific backend
// instance, so switching servers must clear them to avoid leaking credentials
// across instances. Exported so the auth layer (task_f61e) reuses the same keys
// rather than duplicating these strings.
export const SESSION_STORAGE_KEYS = [
  '@cairn:access_token',
  '@cairn:refresh_token',
  '@cairn:token_expires_at',
  '@cairn:user',
] as const;

let storage: StorageAdapter | null = null;
let currentServerUrl: string = DEFAULT_SERVER_URL;

/** Configure the persistence backend. Must be called once during app startup. */
export function configureStorage(adapter: StorageAdapter): void {
  storage = adapter;
}

function getStorage(): StorageAdapter {
  if (!storage) {
    throw new Error('Storage adapter not configured. Call configureStorage() first.');
  }
  return storage;
}

function normalizeServerUrl(url: string): string {
  let trimmed = url.trim().replace(/\/+$/, '');
  if (!trimmed) return DEFAULT_SERVER_URL;
  if (!/^https?:\/\//i.test(trimmed)) {
    const cleanUrl = trimmed.replace(/^https?:\/*/i, '');
    if (!cleanUrl) return DEFAULT_SERVER_URL;
    trimmed = `https://${cleanUrl}`;
  }
  return trimmed;
}

export function getServerUrl(): string {
  return currentServerUrl;
}

export async function loadServerUrl(): Promise<string> {
  try {
    const stored = await getStorage().getItem(SERVER_URL_KEY);
    currentServerUrl = stored ? normalizeServerUrl(stored) : DEFAULT_SERVER_URL;
  } catch (error) {
    console.error('Failed to load server URL, using default:', error);
    currentServerUrl = DEFAULT_SERVER_URL;
  }
  return currentServerUrl;
}

export async function setServerUrl(url: string): Promise<string> {
  const normalized = normalizeServerUrl(url);
  const hasChanged = currentServerUrl !== normalized;
  if (normalized === DEFAULT_SERVER_URL) {
    await getStorage().removeItem(SERVER_URL_KEY);
  } else {
    await getStorage().setItem(SERVER_URL_KEY, normalized);
  }
  currentServerUrl = normalized;
  // Pointing at a different backend invalidates the current session — clear it,
  // but only on a real change so a no-op save doesn't sign the user out.
  if (hasChanged) {
    await Promise.all(SESSION_STORAGE_KEYS.map((key) => getStorage().removeItem(key)));
  }
  return normalized;
}
