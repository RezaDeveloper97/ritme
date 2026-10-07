import { z } from 'zod';

import { INSTRUCTOR_API_BASE_URL, PUBLIC_API_BASE_URL } from '@/shared/config';
import { clearAuthToken, getAuthToken } from '@/shared/session';

import { envelopeSchema, errorBodySchema } from './envelope';
import { ApiError } from './errors';

/**
 * The single thin fetch client for instructor-web. Every call:
 *  - sends `Authorization: Bearer <token>` when a session exists;
 *  - unwraps `{success, data}` and validates `data` with the caller's zod schema;
 *  - turns every failure into an ApiError;
 *  - a JSON 401 from the API drops the token and calls the unauthorized handler
 *    (app/providers → /login). A body-less 401 is the stage Basic-auth gate, not
 *    the API, and must not end the session;
 *  - a 403 `instructor_required` / `instructor_pending` calls the forbidden
 *    handler, so the gate re-reads GET /me and routes to «درخواست» / «در انتظار».
 */

export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions<S extends z.ZodTypeAny> {
  query?: Record<string, QueryValue>;
  body?: unknown;
  schema?: S;
  signal?: AbortSignal;
  /** `instructor` (default) → /api/instructor/v1, `public` → /api/v1. */
  api?: 'instructor' | 'public';
  /** Don't run the unauthorized handler on 401 (e.g. logout of a dead token). */
  skipAuthRedirect?: boolean;
}

type Handler = (error: ApiError) => void;
let onUnauthorized: Handler | null = null;
let onForbidden: Handler | null = null;

export function setUnauthorizedHandler(handler: Handler | null): void {
  onUnauthorized = handler;
}

export function setForbiddenHandler(handler: Handler | null): void {
  onForbidden = handler;
}

export const INSTRUCTOR_FORBIDDEN_CODES: ReadonlySet<string> = new Set([
  'instructor_required',
  'instructor_pending',
]);

export function buildUrl(base: string, path: string, query?: Record<string, QueryValue>): string {
  const url = `${base}${path.startsWith('/') ? path : `/${path}`}`;
  if (!query) return url;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === '') continue;
    params.set(key, String(value));
  }
  const qs = params.toString();
  return qs ? `${url}?${qs}` : url;
}

async function parseJson(res: Response): Promise<unknown> {
  const type = res.headers.get('content-type') ?? '';
  if (!type.includes('json')) return undefined;
  try {
    return await res.json();
  } catch {
    return undefined;
  }
}

export function toApiError(status: number, body: unknown, retryHeader: string | null): ApiError {
  const parsed = errorBodySchema.safeParse(body);
  const err = parsed.success ? parsed.data : {};
  const fromHeader = retryHeader ? Number(retryHeader) : NaN;
  const retryAfter =
    err.retry_after ?? err.data?.retry_after ?? (Number.isFinite(fromHeader) ? fromHeader : null);
  let code = err.error_code;
  if (!code) {
    if (status === 422 && err.errors) code = 'validation_failed';
    else if (status === 404) code = 'not_found';
    else if (status === 429) code = 'too_many_requests';
    else code = 'http_error';
  }
  return new ApiError({ status, code, message: err.message, fieldErrors: err.errors, retryAfter });
}

export async function request<S extends z.ZodTypeAny = z.ZodUnknown>(
  method: Method,
  path: string,
  options: RequestOptions<S> = {},
): Promise<z.output<S>> {
  const base = options.api === 'public' ? PUBLIC_API_BASE_URL : INSTRUCTOR_API_BASE_URL;
  const url = buildUrl(base, path, options.query);

  const headers: Record<string, string> = { Accept: 'application/json', 'Accept-Language': 'fa' };
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  const token = getAuthToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  let res: Response;
  try {
    res = await fetch(url, {
      method,
      headers,
      // Same origin; the stage gate cookie rides along on its own.
      credentials: 'same-origin',
      signal: options.signal,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    });
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause;
    throw new ApiError({ status: 0, code: 'network_error' });
  }

  const body = await parseJson(res);
  if (!res.ok) {
    const error = toApiError(res.status, body, res.headers.get('retry-after'));
    if (res.status === 401 && body !== undefined && !options.skipAuthRedirect) {
      clearAuthToken();
      onUnauthorized?.(error);
    }
    if (res.status === 403 && INSTRUCTOR_FORBIDDEN_CODES.has(error.code)) {
      onForbidden?.(error);
    }
    throw error;
  }

  const schema = (options.schema ?? z.unknown()) as S;
  const parsed = envelopeSchema(schema).safeParse(body);
  if (!parsed.success) {
    throw new ApiError({ status: res.status, code: 'invalid_response', message: parsed.error.message });
  }
  return parsed.data.data as z.output<S>;
}

type Opts<S extends z.ZodTypeAny> = Omit<RequestOptions<S>, 'body'>;

export const api = {
  get: <S extends z.ZodTypeAny = z.ZodUnknown>(path: string, opts?: Opts<S>) => request('GET', path, opts),
  post: <S extends z.ZodTypeAny = z.ZodUnknown>(path: string, body?: unknown, opts?: Opts<S>) =>
    request('POST', path, { ...opts, body }),
  put: <S extends z.ZodTypeAny = z.ZodUnknown>(path: string, body?: unknown, opts?: Opts<S>) =>
    request('PUT', path, { ...opts, body }),
  delete: <S extends z.ZodTypeAny = z.ZodUnknown>(path: string, opts?: Opts<S>) => request('DELETE', path, opts),
};
