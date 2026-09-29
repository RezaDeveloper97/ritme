import { ApiError } from './apiClient';

/**
 * The Ritme API wraps every response in a consistent envelope
 * (`{ success, message, data }`) — see the OpenAPI spec (CLAUDE.md §8.1).
 * Types + helpers here keep slices from re-parsing that shape by hand.
 */
export interface ApiEnvelope<T> {
  success: boolean;
  message?: string;
  data?: T;
  errors?: Record<string, string[]>;
  /** Machine-readable reason on some failures, e.g. `limit_reached`. */
  error_code?: string;
  retry_after?: number;
}

function errorBody(error: unknown): ApiEnvelope<unknown> | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const body = error.response?.data;
  // A non-JSON body (a proxy's HTML page) is a string with nothing to read.
  return typeof body === 'object' && body !== null ? (body as ApiEnvelope<unknown>) : undefined;
}

/** Extracts the server-provided message from a failed request, if any. */
export function getApiErrorMessage(error: unknown): string | undefined {
  return errorBody(error)?.message;
}

/** The envelope's `error_code` of a failed request, if any. */
export function getApiErrorCode(error: unknown): string | undefined {
  const code = errorBody(error)?.error_code;
  return typeof code === 'string' ? code : undefined;
}

/**
 * A per-user cap was hit (422 `error_code: "limit_reached"` — too many
 * medications, appointments or custom checkups). Returns the server's
 * localized message (or its `errors.limit` entry) so forms can show it
 * instead of the generic save error; `undefined` for any other failure.
 */
export function getApiLimitMessage(error: unknown): string | undefined {
  if (getApiErrorStatus(error) !== 422 || getApiErrorCode(error) !== 'limit_reached') return undefined;
  const body = errorBody(error);
  const message = typeof body?.message === 'string' && body.message.trim() ? body.message : undefined;
  const limit = body?.errors?.limit;
  const fromErrors = Array.isArray(limit) ? limit.find((m) => typeof m === 'string' && m.trim()) : undefined;
  return message ?? fromErrors;
}

/**
 * The per-user write limit was hit (429 `error_code: "too_many_requests"` — the
 * Go-only care, checkup and fertility writes). Returns the server's message,
 * already in the request language; `undefined` for any other failure,
 * including a Laravel-style 429 whose body is the framework's English
 * "Too Many Attempts." — callers fall back to their own copy.
 */
export function getApiThrottleMessage(error: unknown): string | undefined {
  if (getApiErrorStatus(error) !== 429 || getApiErrorCode(error) !== 'too_many_requests') return undefined;
  const message = errorBody(error)?.message;
  return typeof message === 'string' && message.trim() ? message : undefined;
}

/**
 * What a save form shows for a failed write: the server's localized cap (422
 * `limit_reached`) or write-throttle (429) message, else `fallback`.
 */
export function getApiSaveErrorMessage(error: unknown, fallback: string): string {
  return getApiLimitMessage(error) ?? getApiThrottleMessage(error) ?? fallback;
}

/** HTTP status of a failed request, when it came from the API. */
export function getApiErrorStatus(error: unknown): number | undefined {
  return error instanceof ApiError ? error.response?.status : undefined;
}
