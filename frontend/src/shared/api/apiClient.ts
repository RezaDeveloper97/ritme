import { bearerOf, clearAuthToken, endsSession, getAuthToken } from '@/shared/session';
import { env } from '@/shared/config';

/**
 * The single shared HTTP client for the whole app (CLAUDE.md §2). Slices import
 * `apiClient` from `@/shared/api`; they never call `fetch` for the API directly.
 *
 * A thin `fetch` wrapper with the slice of axios' surface the app used
 * (`get/post/put/delete`, `params`, `{ data }` results, a rejected error that
 * carries `response.status/data/headers`). Replacing axios saved ~15 KB gzip on
 * every route (perf baseline §1.2, §3 #9); the behaviour below deliberately
 * mirrors what axios did, so callers and the session rules did not change.
 *
 * Privacy (§11): never put health data — cycle dates, symptoms, pregnancy
 * status — into URLs, query strings, headers, or logs.
 */

/** Same budget axios had. A timeout rejects with no `response`. */
export const REQUEST_TIMEOUT_MS = 15_000;

type ParamValue = string | number | boolean | null | undefined;

export interface RequestConfig {
  /** Query string. `null`/`undefined` values are dropped, as axios did. */
  params?: Record<string, ParamValue>;
  /** Overrides {@link REQUEST_TIMEOUT_MS} (e.g. an upload that waits for speech-to-text). */
  timeoutMs?: number;
  /**
   * Upload progress (0–1) of a `FormData` body. When set, the request goes over
   * `XMLHttpRequest` (fetch has no upload progress); everything else — headers,
   * timeout, the error shape, the session rules — is the same.
   */
  onUploadProgress?: (fraction: number) => void;
}

export interface ApiResponse<T> {
  data: T;
  status: number;
  /** Lower-cased header names. */
  headers: Record<string, string>;
}

/** Why a request failed without an HTTP response. */
export type ApiErrorCode = 'timeout' | 'network';

/**
 * A failed API request. `response` exists when the server answered with a
 * non-2xx status (the axios shape: `error.response?.status`, `.data`); it is
 * absent for a network error or a timeout, and then `code` says which.
 */
export class ApiError extends Error {
  readonly response?: ApiResponse<unknown>;
  readonly code?: ApiErrorCode;
  /** The bearer token this request carried, if any (for `endsSession`). */
  readonly sentToken: string | null;

  constructor(
    message: string,
    init: { response?: ApiResponse<unknown>; code?: ApiErrorCode; sentToken: string | null },
  ) {
    super(message);
    this.name = 'ApiError';
    this.response = init.response;
    this.code = init.code;
    this.sentToken = init.sentToken;
  }
}

function buildUrl(path: string, params: RequestConfig['params']): string {
  // axios' baseURL join: exactly one slash between base and path.
  let url = `${env.apiBaseUrl.replace(/\/+$/, '')}/${path.replace(/^\/+/, '')}`;
  if (params) {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null) query.append(key, String(value));
    }
    const qs = query.toString();
    if (qs) url += (url.includes('?') ? '&' : '?') + qs;
  }
  return url;
}

function headersOf(response: Response): Record<string, string> {
  const out: Record<string, string> = {};
  response.headers.forEach((value, key) => {
    out[key.toLowerCase()] = value;
  });
  return out;
}

/**
 * The body as axios read it by default: JSON when it parses, the raw text when
 * it doesn't (a proxy's HTML page), and `''` for an empty body.
 */
async function readBody(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) return '';
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

/** A `FormData` request over XHR for its upload progress; rejects (no response) on a network error or abort. */
function sendWithProgress(
  method: string,
  url: string,
  headers: Record<string, string>,
  body: FormData,
  signal: AbortSignal,
  onProgress: (fraction: number) => void,
): Promise<ApiResponse<unknown>> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open(method, url);
    for (const [key, value] of Object.entries(headers)) xhr.setRequestHeader(key, value);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && e.total > 0) onProgress(Math.min(1, e.loaded / e.total));
    };
    xhr.onload = () => {
      const out: Record<string, string> = {};
      for (const line of xhr.getAllResponseHeaders().trim().split(/[\r\n]+/)) {
        const i = line.indexOf(':');
        if (i > 0) out[line.slice(0, i).trim().toLowerCase()] = line.slice(i + 1).trim();
      }
      let data: unknown = xhr.responseText;
      if (!xhr.responseText) data = '';
      else {
        try {
          data = JSON.parse(xhr.responseText) as unknown;
        } catch {
          // a proxy's HTML page stays text, as readBody() does
        }
      }
      resolve({ data, status: xhr.status, headers: out });
    };
    xhr.onerror = () => reject(new Error('network'));
    xhr.onabort = () => reject(new Error('abort'));
    signal.addEventListener('abort', () => xhr.abort(), { once: true });
    xhr.send(body);
  });
}

async function request<T>(
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  path: string,
  body: unknown,
  config: RequestConfig | undefined,
): Promise<ApiResponse<T>> {
  // axios' default Accept. It matters: Laravel only answers an auth failure
  // with a JSON 401 (the shape `endsSession` trusts) when the request accepts JSON.
  const headers: Record<string, string> = { Accept: 'application/json, text/plain, */*' };

  // Attach the JWT bearer token from the session store (§8.1). Centralized here
  // so auth is never re-implemented per slice.
  const token = getAuthToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  // Tell the API which locale to answer in (§6). The backend localizes
  // messages/phase text off `Accept-Language`; without this the browser's own
  // header (often `en`) leaks through and Persian users get English copy. The
  // active locale lives on `<html lang>` (set per-request by the locale
  // layout); default to the product default `fa` when it's unavailable.
  headers['Accept-Language'] =
    typeof document !== 'undefined' && document.documentElement.lang
      ? document.documentElement.lang
      : 'fa';

  // Like axios: a JSON Content-Type only when there is a body to describe. A
  // FormData body (file upload) goes as is; the browser sets the multipart
  // Content-Type with its boundary.
  const hasBody = body !== undefined;
  const isForm = typeof FormData !== 'undefined' && body instanceof FormData;
  if (hasBody && !isForm) headers['Content-Type'] = 'application/json';

  const sentToken = bearerOf(headers.Authorization);

  // A plain controller + timer rather than `AbortSignal.timeout()`, which iOS
  // Safari only gained in 16 and the app still serves older WebKit.
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, config?.timeoutMs ?? REQUEST_TIMEOUT_MS);

  let result: ApiResponse<unknown>;
  try {
    if (isForm && config?.onUploadProgress && typeof XMLHttpRequest !== 'undefined') {
      result = await sendWithProgress(method, buildUrl(path, config.params), headers, body as FormData, controller.signal, config.onUploadProgress);
    } else {
      const response = await fetch(buildUrl(path, config?.params), {
        method,
        headers,
        body: isForm ? (body as FormData) : hasBody ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });
      // The budget covers the body too, as axios' did.
      result = {
        data: await readBody(response),
        status: response.status,
        headers: headersOf(response),
      };
    }
  } catch {
    // No response at all: never a reason to touch the session.
    throw new ApiError(timedOut ? 'Request timed out' : 'Network error', {
      code: timedOut ? 'timeout' : 'network',
      sentToken,
    });
  } finally {
    clearTimeout(timer);
  }
  if (result.status >= 200 && result.status < 300) return result as ApiResponse<T>;

  // On 401 the token may be gone — but only clear it when the API said so about
  // the token this request carried (`endsSession`, unit-tested in shared/session):
  // a JSON body with `error_code` token_expired/token_revoked/unauthenticated, or
  // Laravel's legacy `Unauthenticated.` body. A proxy's HTML 401 (staging's
  // password gate once did this), network errors, timeouts, 5xx, and a 401 for a
  // request that raced a token refresh all leave the session alone. Route
  // redirection is handled by SessionGuard, not here.
  if (
    endsSession({
      status: result.status,
      contentType: result.headers['content-type'] ?? '',
      body: result.data,
      sentToken,
      currentToken: getAuthToken(),
    })
  ) {
    clearAuthToken();
  }
  throw new ApiError(`Request failed with status code ${result.status}`, {
    response: result,
    sentToken,
  });
}

export const apiClient = {
  get<T = unknown>(path: string, config?: RequestConfig): Promise<ApiResponse<T>> {
    return request<T>('GET', path, undefined, config);
  },
  delete<T = unknown>(path: string, config?: RequestConfig): Promise<ApiResponse<T>> {
    return request<T>('DELETE', path, undefined, config);
  },
  post<T = unknown>(path: string, data?: unknown, config?: RequestConfig): Promise<ApiResponse<T>> {
    return request<T>('POST', path, data, config);
  },
  put<T = unknown>(path: string, data?: unknown, config?: RequestConfig): Promise<ApiResponse<T>> {
    return request<T>('PUT', path, data, config);
  },
  patch<T = unknown>(path: string, data?: unknown, config?: RequestConfig): Promise<ApiResponse<T>> {
    return request<T>('PATCH', path, data, config);
  },
};
