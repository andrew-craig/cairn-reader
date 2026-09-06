/**
 * Thrown when a request could not be completed because the server was
 * unreachable (network failure, timeout, 5xx response, or a malformed/
 * unparseable body) — as opposed to a definitive rejection (4xx).
 *
 * Distinguishing the two matters: retry.ts treats unrecognized errors as
 * retryable, and callers must not treat "couldn't reach the server" the same
 * as "server said no" (e.g. by logging the user out). Keep this message free
 * of the substrings retry.ts uses to classify non-retryable auth failures
 * ('session expired', 'not authenticated', 'unauthorized', 'forbidden',
 * 'not found', 'bad request').
 */
export class NetworkError extends Error {
  constructor(message: string = 'Unable to reach the server. Please try again later.') {
    super(message);
    this.name = 'NetworkError';
    Object.setPrototypeOf(this, NetworkError.prototype);
  }
}
