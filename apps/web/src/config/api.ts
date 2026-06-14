// Server URL configuration for the web app.
//
// Unlike the mobile app (apps/mobile/src/config/api.ts), which ships against a
// fixed default backend, the web app is served from the same origin as the API
// (selfhost single binary, or behind Caddy in prod). It therefore defaults to
// its own origin and builds relative requests against it, while still allowing a
// runtime override (e.g. pointing a locally served build at a remote backend).
//
// Persistence is injected via a StorageAdapter so the browser can supply
// localStorage (see decision_9b2d ADR).

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

/** The origin the app is served from — the default backend for same-origin deployments. */
export function getDefaultServerUrl(): string {
  return typeof window !== 'undefined' ? window.location.origin : '';
}

let storage: StorageAdapter | null = null;
let currentServerUrl: string = getDefaultServerUrl();

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
