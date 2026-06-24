// Startup side effects: wire the shared API config layer to mobile's persistence
// and default backend (decision_9b2d ADR). Imported as the very first thing in
// index.js so this runs before any service/component module that might read the
// server URL is evaluated.
import { configureStorage, configureDefaultServerUrl } from '@cairn/shared';

import { asyncStorageAdapter, DEFAULT_SERVER_URL } from './storage';

configureStorage(asyncStorageAdapter);
configureDefaultServerUrl(() => DEFAULT_SERVER_URL);
