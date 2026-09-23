import { z } from 'zod';

import { ADMIN_API_BASE_URL, PUBLIC_API_BASE_URL } from '@/shared/config';

import { getCsrfToken, setCsrfToken } from './csrf';
import { envelopeSchema, errorBodySchema } from './envelope';
import { ApiError } from './errors';

/**
 * The single thin fetch client for admin-web. Every call:
 *  - sends the session cookie (`credentials: 'include'`, needed for cross-origin dev);
 *  - adds X-CSRF-Token on POST/PUT/PATCH/DELETE, and on a 419 refreshes the token
 *    from GET /auth/me and retries once;
 *  - unwraps `{success, data}` and validates `data` with the caller's zod schema;
 *  - turns every failure into an ApiError; a 401 also calls the unauthorized
 *    handler (installed by app/providers → redirect to /login).
 */

export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions<S extends z.ZodTypeAny> {
  query?: Record<string, QueryValue>;
  /** JSON-serialised, or sent as-is when FormData (multipart uploads). */
  body?: unknown;
  schema?: S;
  signal?: AbortSignal;
  /** `admin` (default) → /api/admin/v1, `public` → /api/v1. */
  api?: 'admin' | 'public';
  /** Don't run the unauthorized handler on 401 (the login page's own probe). */
  skipAuthRedirect?: boolean;
}

type UnauthorizedHandler = (error: ApiError) => void;
let onUnauthorized: UnauthorizedHandler | null = null;

export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  onUnauthorized = handler;
}

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

const MUTATING: ReadonlySet<Method> = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

async function parseJson(res: Response): Promise<unknown> {
  const type = res.headers.get('content-type') ?? '';
  if (!type.includes('json')) return undefined;
  try {
    return await res.json();
  } catch {
    return undefined;
  }
}

function toApiError(status: number, body: unknown, retryHeader: string | null): ApiError {
  const parsed = errorBodySchema.safeParse(body);
  const err = parsed.success ? parsed.data : {};
  const fromHeader = retryHeader ? Number(retryHeader) : NaN;
  return new ApiError({
    status,
    code: err.error_code ?? (status === 404 ? 'not_found' : 'http_error'),
    message: err.message,
    fieldErrors: err.errors,
    retryAfter: err.retry_after ?? (Number.isFinite(fromHeader) ? fromHeader : null),
  });
}

const csrfRefreshSchema = envelopeSchema(z.object({ csrf_token: z.string() }).passthrough());

async function refreshCsrf(): Promise<boolean> {
  try {
    const res = await fetch(buildUrl(ADMIN_API_BASE_URL, '/auth/me'), {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    });
    const parsed = csrfRefreshSchema.safeParse(await parseJson(res));
    if (!res.ok || !parsed.success) return false;
    setCsrfToken(parsed.data.data.csrf_token);
    return true;
  } catch {
    return false;
  }
}

export async function request<S extends z.ZodTypeAny = z.ZodUnknown>(
  method: Method,
  path: string,
  options: RequestOptions<S> = {},
): Promise<z.output<S>> {
  const base = options.api === 'public' ? PUBLIC_API_BASE_URL : ADMIN_API_BASE_URL;
  const url = buildUrl(base, path, options.query);
  const isForm = typeof FormData !== 'undefined' && options.body instanceof FormData;

  const send = async (): Promise<Response> => {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (options.body !== undefined && !isForm) headers['Content-Type'] = 'application/json';
    if (MUTATING.has(method)) {
      const csrf = getCsrfToken();
      if (csrf) headers['X-CSRF-Token'] = csrf;
    }
    try {
      return await fetch(url, {
        method,
        headers,
        credentials: 'include',
        signal: options.signal,
        body:
          options.body === undefined
            ? undefined
            : isForm
              ? (options.body as FormData)
              : JSON.stringify(options.body),
      });
    } catch (cause) {
      if (cause instanceof DOMException && cause.name === 'AbortError') throw cause;
      throw new ApiError({ status: 0, code: 'network_error' });
    }
  };

  let res = await send();
  if (res.status === 419 && MUTATING.has(method) && (await refreshCsrf())) {
    res = await send();
  }

  const body = await parseJson(res);
  if (!res.ok) {
    const error = toApiError(res.status, body, res.headers.get('retry-after'));
    // Only a JSON 401 from the API ends the session (a proxy/Basic-auth 401 has no body).
    if (res.status === 401 && body !== undefined && !options.skipAuthRedirect) {
      setCsrfToken(null);
      onUnauthorized?.(error);
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
  patch: <S extends z.ZodTypeAny = z.ZodUnknown>(path: string, body?: unknown, opts?: Opts<S>) =>
    request('PATCH', path, { ...opts, body }),
  delete: <S extends z.ZodTypeAny = z.ZodUnknown>(path: string, opts?: Opts<S>) => request('DELETE', path, opts),
};
