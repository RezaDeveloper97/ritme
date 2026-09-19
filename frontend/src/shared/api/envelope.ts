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
  retry_after?: number;
}

/** Extracts the server-provided message from a failed request, if any. */
export function getApiErrorMessage(error: unknown): string | undefined {
  if (error instanceof ApiError) {
    const body = error.response?.data as ApiEnvelope<unknown> | undefined;
    // A non-JSON body (a proxy's HTML page) is a string with no message.
    return typeof body === 'object' && body !== null ? body.message : undefined;
  }
  return undefined;
}

/** HTTP status of a failed request, when it came from the API. */
export function getApiErrorStatus(error: unknown): number | undefined {
  return error instanceof ApiError ? error.response?.status : undefined;
}
