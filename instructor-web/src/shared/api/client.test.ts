import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { z } from 'zod';

import { clearAuthToken, setAuthToken } from '@/shared/session';

import { api, setForbiddenHandler, setUnauthorizedHandler, toApiError } from './client';
import { ApiError } from './errors';

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });
}

const fetchMock = vi.fn<typeof fetch>();

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock);
  fetchMock.mockReset();
  clearAuthToken();
});
afterEach(() => {
  vi.unstubAllGlobals();
  setUnauthorizedHandler(null);
  setForbiddenHandler(null);
});

describe('api client', () => {
  it('sends the bearer token to the instructor API and unwraps the envelope', async () => {
    setAuthToken('tok-1');
    fetchMock.mockResolvedValue(jsonResponse(200, { success: true, data: { instructor: null } }));
    const data = await api.get('/me', { schema: z.object({ instructor: z.null() }) });
    expect(data).toEqual({ instructor: null });
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe('/api/instructor/v1/me');
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok-1');
  });

  it('uses /api/v1 for the public API and no Authorization without a session', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { success: true, data: { expires_in: 120 } }));
    await api.post('/auth/send-otp', { mobile: '09120000000' }, { api: 'public' });
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe('/api/v1/auth/send-otp');
    expect((init?.headers as Record<string, string>).Authorization).toBeUndefined();
    expect(init?.body).toBe('{"mobile":"09120000000"}');
  });

  it('a JSON 401 clears the token and calls the unauthorized handler', async () => {
    setAuthToken('dead');
    const onUnauthorized = vi.fn();
    setUnauthorizedHandler(onUnauthorized);
    fetchMock.mockResolvedValue(jsonResponse(401, { success: false, message: 'Unauthenticated.' }));
    await expect(api.get('/me')).rejects.toBeInstanceOf(ApiError);
    expect(onUnauthorized).toHaveBeenCalledOnce();
    fetchMock.mockResolvedValue(jsonResponse(200, { success: true, data: {} }));
    await api.get('/me');
    expect((fetchMock.mock.calls[1]![1]?.headers as Record<string, string>).Authorization).toBeUndefined();
  });

  it('a body-less 401 (stage Basic-auth gate) keeps the session', async () => {
    setAuthToken('alive');
    const onUnauthorized = vi.fn();
    setUnauthorizedHandler(onUnauthorized);
    fetchMock.mockResolvedValue(new Response('', { status: 401, headers: { 'content-type': 'text/html' } }));
    await expect(api.get('/me')).rejects.toMatchObject({ status: 401 });
    expect(onUnauthorized).not.toHaveBeenCalled();
  });

  it('403 instructor_pending calls the forbidden handler with the code', async () => {
    const onForbidden = vi.fn();
    setForbiddenHandler(onForbidden);
    fetchMock.mockResolvedValue(
      jsonResponse(403, { success: false, message: 'x', error_code: 'instructor_pending' }),
    );
    await expect(api.get('/courses')).rejects.toMatchObject({ code: 'instructor_pending', status: 403 });
    expect(onForbidden).toHaveBeenCalledOnce();
  });

  it('maps network failures and invalid envelopes', async () => {
    fetchMock.mockRejectedValue(new TypeError('offline'));
    await expect(api.get('/me')).rejects.toMatchObject({ code: 'network_error', status: 0 });
    fetchMock.mockResolvedValue(jsonResponse(200, { nope: true }));
    await expect(api.get('/me')).rejects.toMatchObject({ code: 'invalid_response' });
  });
});

describe('toApiError', () => {
  it('reads send-otp retry_after from data', () => {
    const e = toApiError(429, { success: false, message: 'wait', data: { retry_after: 42.5 } }, null);
    expect(e.retryAfter).toBe(42.5);
    expect(e.code).toBe('too_many_requests');
  });
  it('flags 422 field errors', () => {
    const e = toApiError(422, { success: false, errors: { display_name: ['required'] } }, null);
    expect(e.code).toBe('validation_failed');
    expect(e.field('display_name')).toBe('required');
  });
});
