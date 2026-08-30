/**
 * Retry utility with exponential backoff for transient failures.
 *
 * Retries on:
 *   - Network / abort errors (TypeError: network request failed, AbortError, …)
 *   - HTTP 5xx responses (surfaced as an HttpResponseError by the caller)
 *
 * Never retries 4xx responses, and never retries the client-side auth failures
 * thrown before any response exists ("Session expired…", "Not authenticated").
 */

const DEFAULT_MAX_RETRIES = 3;
const BASE_DELAY_MS = 1000;
const DEFAULT_TIMEOUT_MS = 15000;

/**
 * Carries the HTTP status of a failed response so the retry decision can key off
 * the status rather than the response's prose. Callers that want a 5xx retried
 * should throw this from inside the withRetry callback.
 */
export class HttpResponseError extends Error {
  readonly status: number;

  constructor(status: number, message?: string) {
    super(message ?? `HTTP ${status}`);
    this.name = 'HttpResponseError';
    this.status = status;
  }
}

// Auth failures raised before a response exists (no status to branch on). These
// are unrecoverable by a retry — the caller must re-authenticate. The exact
// strings are produced by the shared AuthService.fetchWithAuth.
const NON_RETRYABLE_CLIENT_MESSAGES = ['session expired', 'not authenticated'];

function statusOf(error: unknown): number | undefined {
  if (error instanceof Error) {
    const status = (error as { status?: unknown }).status;
    if (typeof status === 'number') return status;
  }
  return undefined;
}

/**
 * Returns true for errors that are safe to retry (network / abort / 5xx).
 *
 * Decision order:
 *   1. An error carrying an HTTP status → retry iff it is 5xx. 4xx never retries.
 *   2. A client-side auth failure thrown before any response → never retry.
 *   3. Anything else (network failure, abort, unknown) → retry.
 */
function isRetryable(error: unknown): boolean {
  const status = statusOf(error);
  if (status !== undefined) {
    return status >= 500 && status <= 599;
  }

  if (error instanceof Error) {
    const msg = error.message.toLowerCase();
    if (NON_RETRYABLE_CLIENT_MESSAGES.some((m) => msg.includes(m))) {
      return false;
    }
  }

  return true;
}

/**
 * Wraps a fetch call with:
 *   - per-request AbortController timeout (default 15 s)
 *   - automatic retry with exponential backoff for 5xx / network errors
 *
 * @param fn    Async function that performs the request. Receives an AbortSignal.
 * @param opts  Optional overrides for retries and timeout.
 */
export async function withRetry<T>(
  fn: (signal: AbortSignal) => Promise<T>,
  opts?: { maxRetries?: number; timeoutMs?: number }
): Promise<T> {
  const maxRetries = opts?.maxRetries ?? DEFAULT_MAX_RETRIES;
  const timeoutMs = opts?.timeoutMs ?? DEFAULT_TIMEOUT_MS;

  let lastError: unknown;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    const controller = new AbortController();
    const timerId = setTimeout(() => controller.abort(), timeoutMs);

    try {
      const result = await fn(controller.signal);
      clearTimeout(timerId);
      return result;
    } catch (err) {
      clearTimeout(timerId);
      lastError = err;

      const isLast = attempt === maxRetries;
      if (isLast || !isRetryable(err)) {
        throw err;
      }

      // Exponential backoff: 1s, 2s, 4s, …
      const delay = BASE_DELAY_MS * Math.pow(2, attempt);
      await new Promise<void>((resolve) => setTimeout(resolve, delay));
    }
  }

  throw lastError;
}
