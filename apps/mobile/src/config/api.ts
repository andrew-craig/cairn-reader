import AsyncStorage from '@react-native-async-storage/async-storage';

export const DEFAULT_SERVER_URL = 'https://cairn.seatrain.net';
const SERVER_URL_KEY = '@cairn:server_url';

let currentServerUrl: string = DEFAULT_SERVER_URL;

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
    const stored = await AsyncStorage.getItem(SERVER_URL_KEY);
    currentServerUrl = stored ? normalizeServerUrl(stored) : DEFAULT_SERVER_URL;
  } catch (error) {
    console.error('Failed to load server URL, using default:', error);
    currentServerUrl = DEFAULT_SERVER_URL;
  }
  return currentServerUrl;
}

export async function setServerUrl(url: string): Promise<string> {
  const normalized = normalizeServerUrl(url);
  if (normalized === DEFAULT_SERVER_URL) {
    await AsyncStorage.removeItem(SERVER_URL_KEY);
  } else {
    await AsyncStorage.setItem(SERVER_URL_KEY, normalized);
  }
  currentServerUrl = normalized;
  return normalized;
}
