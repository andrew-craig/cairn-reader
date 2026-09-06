/**
 * Retry utility with exponential backoff for transient failures.
 *
 * Retries on:
 *   - Network errors (TypeError: network request failed, AbortError, etc.)
 *   - HTTP 5xx responses (when the caller throws on those)
 *
 * Does NOT retry on 4xx client errors.
 */

import { NetworkError } from './errors';

const DEFAULT_MAX_RETRIES = 3;
const BASE_DELAY_MS = 1000;
const DEFAULT_TIMEOUT_MS = 15000;

/**
 * Returns true for errors that are safe to retry (network / server transients).
 * 4xx errors are NOT retried — they indicate a client-side problem.
 */
function isRetryable(error: unknown): boolean {
  if (!(error instanceof Error)) return true;

  // A NetworkError means "couldn't reach the server" by construction — always
  // retryable, regardless of what its message happens to contain.
  if (error instanceof NetworkError) return true;

  const msg = error.message.toLowerCase();

  // 4xx client errors — do not retry
  if (
    msg.includes('not authenticated') ||
    msg.includes('session expired') ||
    msg.includes('unauthorized') ||
    msg.includes('forbidden') ||
    msg.includes('not found') ||
    msg.includes('bad request')
  ) {
    return false;
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
