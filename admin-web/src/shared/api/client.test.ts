import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { z } from 'zod';

import { api, buildUrl, setUnauthorizedHandler } from './client';
import { setCsrfToken } from './csrf';
import { ApiError } from './errors';

type Call = { url: string; init: RequestInit };
let calls: Call[];
let responses: Array<() => Response>;

const json = (status: number, body: unknown, headers: Record<string, string> = {}) => () =>
  new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json', ...headers } });

beforeEach(() => {
  calls = [];
  responses = [];
  setCsrfToken(null);
  setUnauthorizedHandler(null);
  vi.stubGlobal('fetch', async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const next = responses.shift();
    if (!next) throw new TypeError('network down');
    return next();
  });
});
afterEach(() => vi.unstubAllGlobals());

const headersOf = (c: Call) => new Headers(c.init.headers);

describe('buildUrl', () => {
  it('joins base and path and drops empty query values', () => {
    expect(buildUrl('/api/admin/v1', '/users', { page: 2, q: '', status: 'all', x: null })).toBe(
      '/api/admin/v1/users?page=2&status=all',
    );
    expect(buildUrl('/api/v1', 'languages')).toBe('/api/v1/languages');
  });
});

describe('request', () => {
  it('unwraps and validates data, always with credentials', async () => {
    responses.push(json(200, { success: true, data: { n: 3 } }));
    await expect(api.get('/x', { schema: z.object({ n: z.number() }) })).resolves.toEqual({ n: 3 });
    expect(calls[0]!.init.credentials).toBe('include');
    expect(calls[0]!.url).toBe('/api/admin/v1/x');
  });

  it('uses the public base for api: public', async () => {
    responses.push(json(200, { success: true, data: null }));
    await api.get('/languages', { api: 'public' });
    expect(calls[0]!.url).toBe('/api/v1/languages');
  });

  it('sends X-CSRF-Token on writes only', async () => {
    setCsrfToken('tok');
    responses.push(json(200, { success: true, data: null }), json(200, { success: true, data: null }));
    await api.get('/a');
    await api.post('/b', { a: 1 });
    expect(headersOf(calls[0]!).get('X-CSRF-Token')).toBeNull();
    expect(headersOf(calls[1]!).get('X-CSRF-Token')).toBe('tok');
    expect(headersOf(calls[1]!).get('Content-Type')).toBe('application/json');
    expect(calls[1]!.init.body).toBe('{"a":1}');
  });

  it('does not set Content-Type for FormData uploads', async () => {
    setCsrfToken('tok');
    responses.push(json(201, { success: true, data: { id: 1 } }));
    const fd = new FormData();
    fd.append('title[fa]', 'سلام');
    await api.post('/banners', fd);
    expect(headersOf(calls[0]!).get('Content-Type')).toBeNull();
    expect(calls[0]!.init.body).toBe(fd);
  });

  it('on 419 refreshes the token from /auth/me and retries once', async () => {
    setCsrfToken('old');
    responses.push(
      json(419, { success: false, error_code: 'csrf_mismatch', message: 'CSRF token mismatch.' }),
      json(200, { success: true, data: { admin: {}, csrf_token: 'new' } }),
      json(200, { success: true, data: { ok: true } }),
    );
    await expect(api.put('/users/1', {})).resolves.toEqual({ ok: true });
    expect(calls.map((c) => c.url)).toEqual(['/api/admin/v1/users/1', '/api/admin/v1/auth/me', '/api/admin/v1/users/1']);
    expect(headersOf(calls[2]!).get('X-CSRF-Token')).toBe('new');
  });

  it('maps a 422 to an ApiError with field errors', async () => {
    responses.push(
      json(422, {
        success: false,
        error_code: 'validation_failed',
        message: 'x',
        errors: { 'title.fa': ['الزامی است'] },
      }),
    );
    const err = await api.post('/articles', {}).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe('validation_failed');
    expect((err as ApiError).field('title.fa')).toBe('الزامی است');
  });

  it('reads retry_after on 429', async () => {
    responses.push(json(429, { success: false, error_code: 'too_many_attempts', retry_after: 42 }, { 'retry-after': '42' }));
    const err = (await api.post('/auth/login', {}).catch((e: unknown) => e)) as ApiError;
    expect(err.retryAfter).toBe(42);
  });

  it('runs the unauthorized handler on a JSON 401', async () => {
    const handler = vi.fn();
    setUnauthorizedHandler(handler);
    responses.push(json(401, { success: false, error_code: 'session_expired' }));
    await expect(api.get('/dashboard')).rejects.toMatchObject({ status: 401, code: 'session_expired' });
    expect(handler).toHaveBeenCalledOnce();
  });

  it('ignores a non-JSON 401 (proxy / Basic auth) and skipAuthRedirect probes', async () => {
    const handler = vi.fn();
    setUnauthorizedHandler(handler);
    responses.push(() => new Response('Unauthorized', { status: 401, headers: { 'content-type': 'text/plain' } }));
    responses.push(json(401, { success: false, error_code: 'unauthenticated' }));
    await expect(api.get('/a')).rejects.toBeInstanceOf(ApiError);
    await expect(api.get('/auth/me', { skipAuthRedirect: true })).rejects.toBeInstanceOf(ApiError);
    expect(handler).not.toHaveBeenCalled();
  });

  it('turns a fetch failure into network_error and a bad body into invalid_response', async () => {
    await expect(api.get('/a')).rejects.toMatchObject({ code: 'network_error', status: 0 });
    responses.push(json(200, { success: true, data: { n: 'x' } }));
    await expect(api.get('/a', { schema: z.object({ n: z.number() }) })).rejects.toMatchObject({
      code: 'invalid_response',
    });
  });
});
