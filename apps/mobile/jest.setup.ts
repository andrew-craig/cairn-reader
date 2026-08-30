// Wire the shared config/service layer for tests, mirroring src/config/init.ts.
// AuthService (now in @cairn/shared) reads its persistence backend via
// getStorage(), which throws until configureStorage() has been called.
import { configureStorage, configureDefaultServerUrl } from '@cairn/shared';
import { asyncStorageAdapter, DEFAULT_SERVER_URL } from './src/config/storage';

configureStorage(asyncStorageAdapter);
configureDefaultServerUrl(() => DEFAULT_SERVER_URL);
