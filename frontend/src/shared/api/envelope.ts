import { AxiosError } from 'axios';

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
  if (error instanceof AxiosError) {
    const body = error.response?.data as ApiEnvelope<unknown> | undefined;
    return body?.message;
  }
  return undefined;
}

/** HTTP status of a failed request, when it came from the API. */
export function getApiErrorStatus(error: unknown): number | undefined {
  return error instanceof AxiosError ? error.response?.status : undefined;
}
