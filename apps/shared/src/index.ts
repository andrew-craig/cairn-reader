// Public surface of the @cairn/shared package: framework-agnostic code reused by
// both apps/mobile and apps/web — the data-model types, the API server-URL /
// persistence layer, and the authentication / token-refresh state machine. The
// latter two are parameterized over an injected default URL and a
// StorageAdapter so each app supplies its own backend (web: serving origin +
// localStorage; mobile: fixed backend + AsyncStorage).
export * from './types';
export * from './config/api';
export * from './services/auth';
export * from './utils/errors';
export * from './utils/http';
export * from './utils/throttle';
