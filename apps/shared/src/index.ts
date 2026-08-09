// Public surface of the @cairn/shared package: framework-agnostic code reused by
// both apps/mobile and apps/web — the data-model types and the API server-URL /
// persistence layer. The latter is parameterized over an injected default URL
// and a StorageAdapter so each app supplies its own backend (web: serving origin
// + localStorage; mobile: fixed backend + AsyncStorage).
export * from './types';
export * from './config/api';
export * from './utils/throttle';
