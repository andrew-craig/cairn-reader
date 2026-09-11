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

/**
 * Thrown when the server responded with a definitive non-2xx status, as
 * opposed to NetworkError's "couldn't reach the server at all". Carries the
 * HTTP status so callers (the outbox drain, task_ebf1) can tell a definitive
 * rejection (4xx) apart from a transient one (401/5xx) without parsing the
 * message text. An error-type change only — the message text callers already
 * key off of (see NetworkError's doc comment, and retry.ts's classification)
 * is unchanged.
 */
export class HttpError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'HttpError';
    this.status = status;
    Object.setPrototypeOf(this, HttpError.prototype);
  }
}
