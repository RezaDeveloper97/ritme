/**
 * Every failure the client surfaces is an ApiError. `code` is the Go admin
 * API's `error_code` (docs/go-migration/admin-api.md §2) or one of the client's
 * own codes below; the UI translates it via the `errors` message namespace and
 * never shows the English `message`.
 */
export const CLIENT_ERROR_CODES = ['network_error', 'invalid_response', 'http_error'] as const;

export type FieldErrors = Record<string, string[]>;

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fieldErrors: FieldErrors;
  readonly retryAfter: number | null;

  constructor(init: {
    status: number;
    code: string;
    message?: string;
    fieldErrors?: FieldErrors;
    retryAfter?: number | null;
  }) {
    super(init.message || init.code);
    this.name = 'ApiError';
    this.status = init.status;
    this.code = init.code;
    this.fieldErrors = init.fieldErrors ?? {};
    this.retryAfter = init.retryAfter ?? null;
  }

  /** First server message for a field (`title.fa`, `email`), if any. */
  field(name: string): string | undefined {
    return this.fieldErrors[name]?.[0];
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

/** 401 is only ever an authentication answer (admin-api.md §2). */
export function isAuthError(error: unknown): boolean {
  return isApiError(error) && error.status === 401;
}
