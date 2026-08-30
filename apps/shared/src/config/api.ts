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

// Persisted session keys. Tokens are issued by (and valid for) a specific backend
// instance, so switching servers must clear them to avoid leaking credentials
// across instances. Exported so the auth layer reuses the same keys rather than
// duplicating these strings.
export const SESSION_STORAGE_KEYS = [
  '@cairn:access_token',
  '@cairn:refresh_token',
  '@cairn:token_expires_at',
  '@cairn:user',
] as const;

let storage: StorageAdapter | null = null;
let resolveDefaultServerUrl: (() => string) | null = null;
let currentServerUrl = '';

/** Configure the persistence backend. Must be called once during app startup. */
export function configureStorage(adapter: StorageAdapter): void {
  storage = adapter;
}

/**
 * Inject the app-specific default server URL resolver and seed the current
 * server URL from it. Must be called once during app startup. The seed only
 * applies on first configuration so a dev hot-reload re-running this does not
 * clobber a server URL already loaded from storage or chosen by the user.
 */
export function configureDefaultServerUrl(resolver: () => string): void {
  resolveDefaultServerUrl = resolver;
  if (!currentServerUrl) {
    currentServerUrl = resolver();
  }
}

export function getDefaultServerUrl(): string {
  if (!resolveDefaultServerUrl) {
    throw new Error(
      'Default server URL resolver not configured. Call configureDefaultServerUrl() first.',
    );
  }
  return resolveDefaultServerUrl();
}

export function getStorage(): StorageAdapter {
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
  const hasChanged = currentServerUrl !== normalized;
  if (normalized === getDefaultServerUrl()) {
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
