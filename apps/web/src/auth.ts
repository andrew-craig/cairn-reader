// Minimal authentication check used only for the top-level route redirect.
// The real auth state (AuthContext, token refresh, login/logout) is implemented
// in a later task (task_f61e); this placeholder simply treats the presence of a
// persisted access token as an authenticated session.
const ACCESS_TOKEN_KEY = '@cairn:access_token';

export function isAuthenticated(): boolean {
  return Boolean(localStorage.getItem(ACCESS_TOKEN_KEY));
}
