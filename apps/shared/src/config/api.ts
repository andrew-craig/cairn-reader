// Server-URL state and persistence for the Cairn API client, shared by
// apps/mobile and apps/web (decision_9b2d ADR — single source of truth).
//
// Two things are app-specific and therefore injected rather than hardcoded here:
//   - the *default* server URL (mobile ships against a fixed backend; web
//     defaults to its serving origin) — supplied via configureDefaultServerUrl.
//   - the persistence backend (AsyncStorage on mobile, localStorage on web) —
//     supplied as a StorageAdapter via configureStorage.
// Both must be configured once during app startup before getServerUrl/
// loadServerUrl/setServerUrl are used.

export interface StorageAdapter {
  getItem(key: string): Promise<string | null>;
  setItem(key: string, value: string): Promise<void>;
  removeItem(key: string): Promise<void>;
}

const SERVER_URL_KEY = '@cairn:server_url';

let storage: StorageAdapter | null = null;
let resolveDefaultServerUrl: () => string = () => '';
let currentServerUrl = '';

/** Configure the persistence backend. Must be called once during app startup. */
export function configureStorage(adapter: StorageAdapter): void {
  storage = adapter;
}

/**
 * Inject the app-specific default server URL resolver and initialize the current
 * server URL to it. Must be called once during app startup.
 */
export function configureDefaultServerUrl(resolver: () => string): void {
  resolveDefaultServerUrl = resolver;
  currentServerUrl = resolver();
}

export function getDefaultServerUrl(): string {
  return resolveDefaultServerUrl();
}

function getStorage(): StorageAdapter {
  if (!storage) {
    throw new Error('Storage adapter not configured. Call configureStorage() first.');
  }
  return storage;
}

function normalizeServerUrl(url: string): string {
  let trimmed = url.trim().replace(/\/+$/, '');
  if (!trimmed) return getDefaultServerUrl();
  if (!/^https?:\/\//i.test(trimmed)) {
    const cleanUrl = trimmed.replace(/^https?:\/*/i, '');
    if (!cleanUrl) return getDefaultServerUrl();
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
    currentServerUrl = stored ? normalizeServerUrl(stored) : getDefaultServerUrl();
  } catch (error) {
    console.error('Failed to load server URL, using default:', error);
    currentServerUrl = getDefaultServerUrl();
  }
  return currentServerUrl;
}

export async function setServerUrl(url: string): Promise<string> {
  const normalized = normalizeServerUrl(url);
  if (normalized === getDefaultServerUrl()) {
    await getStorage().removeItem(SERVER_URL_KEY);
  } else {
    await getStorage().setItem(SERVER_URL_KEY, normalized);
  }
  currentServerUrl = normalized;
  return normalized;
}
