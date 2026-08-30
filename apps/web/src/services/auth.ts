// The web/mobile auth + token-refresh state machine now lives in @cairn/shared
// (parameterized over an injected StorageAdapter — web wires localStorage in
// main.tsx via configureStorage). Web has no login flavor beyond email/password,
// so it consumes the shared class directly.
export { AuthService, RefreshNetworkError, RefreshRejectedError } from '@cairn/shared';
