import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * The fetch client against a minimal fake browser (a Map behind localStorage,
 * a cookie sink, `<html lang>`) and a scripted `fetch`. It pins the behaviour
 * that used to come from axios and, above all, the session rules of CLAUDE.md
 * §11.1: only `endsSession()` may clear the token.
 */
const TOKEN = 'eyJhbGciOi.eyJzdWIiOiIxIn0.c2ln';
const BASE = 'https://api.ritme.app/api/v1';

type FetchImpl = (input: string, init: RequestInit) => Promise<Response>;

function fakeBrowser(fetchImpl: FetchImpl) {
  const store = new Map<string, string>([['ritme_token', TOKEN]]);
  const fetchSpy = vi.fn<FetchImpl>((input, init) =>
    // The same-origin flag route that token.ts pings is not the API.
    input.startsWith('/api/session/') ? Promise.resolve(new Response(null, { status: 204 })) : fetchImpl(input, init),
  );
  vi.stubGlobal('window', {
    localStorage: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
      removeItem: (k: string) => void store.delete(k),
    },
    dispatchEvent: vi.fn(),
  });
  vi.stubGlobal('document', { cookie: '', documentElement: { lang: 'en' } });
  vi.stubGlobal('fetch', fetchSpy);
  return { store, fetchSpy };
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

async function load() {
  vi.resetModules();
  return import('./index');
}

function apiCalls(spy: ReturnType<typeof fakeBrowser>['fetchSpy']) {
  return spy.mock.calls.filter(([url]) => url.startsWith(BASE));
}

describe('apiClient', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  describe('requests', () => {
    let env: ReturnType<typeof fakeBrowser>;
    beforeEach(() => {
      env = fakeBrowser(() => Promise.resolve(json(200, { success: true, data: { ok: 1 } })));
    });

    it('sends the bearer token, the page locale and a JSON Accept on a GET', async () => {
      const { apiClient } = await load();
      const { data, status } = await apiClient.get<{ data: { ok: number } }>('/cycle/today');

      expect(status).toBe(200);
      expect(data.data.ok).toBe(1);
      const [url, init] = apiCalls(env.fetchSpy)[0];
      expect(url).toBe(`${BASE}/cycle/today`);
      const headers = init.headers as Record<string, string>;
      expect(init.method).toBe('GET');
      expect(headers.Authorization).toBe(`Bearer ${TOKEN}`);
      expect(headers['Accept-Language']).toBe('en');
      expect(headers.Accept).toContain('application/json');
      // No body, no Content-Type (as axios did).
      expect(headers['Content-Type']).toBeUndefined();
      expect(init.body).toBeUndefined();
    });

    it('serialises params and drops null/undefined values', async () => {
      const { apiClient } = await load();
      await apiClient.get('/articles', { params: { page: 2, q: 'a b', category: undefined, x: null } });
      expect(apiCalls(env.fetchSpy)[0][0]).toBe(`${BASE}/articles?page=2&q=a+b`);
    });

    it('sends a JSON body on POST/PUT and none on a bare POST', async () => {
      const { apiClient } = await load();
      await apiClient.post('/profile', { weight: 60 });
      await apiClient.put('/cycle/period/7', { start_date: 's', end_date: null });
      await apiClient.post('/cycle/recalculate');

      const [post, put, bare] = apiCalls(env.fetchSpy).map(([, init]) => init);
      expect(post.body).toBe('{"weight":60}');
      expect((post.headers as Record<string, string>)['Content-Type']).toBe('application/json');
      expect(put.method).toBe('PUT');
      expect(put.body).toBe('{"start_date":"s","end_date":null}');
      expect(bare.body).toBeUndefined();
    });

    it('sends a FormData body as is, without a JSON Content-Type (B-N3-05 upload)', async () => {
      const { apiClient } = await load();
      const form = new FormData();
      form.append('duration_ms', '7000');
      await apiClient.post('/logs/voice', form, { timeoutMs: 45_000 });
      const [, init] = apiCalls(env.fetchSpy)[0];
      expect(init.body).toBe(form);
      expect((init.headers as Record<string, string>)['Content-Type']).toBeUndefined();
    });

    it('omits Authorization when signed out', async () => {
      env.store.clear();
      const { apiClient } = await load();
      await apiClient.post('/auth/send-otp', { mobile: 'x' });
      const headers = apiCalls(env.fetchSpy)[0][1].headers as Record<string, string>;
      expect(headers.Authorization).toBeUndefined();
    });
  });

  describe('failures and the session (CLAUDE.md §11.1)', () => {
    it('clears the token on a JSON 401 about the token it sent', async () => {
      const env = fakeBrowser(() =>
        Promise.resolve(json(401, { message: 'Unauthenticated.', error_code: 'token_expired' })),
      );
      const { apiClient, ApiError, getApiErrorStatus } = await load();

      const error = await apiClient.get('/profile').catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect(getApiErrorStatus(error)).toBe(401);
      expect(env.store.has('ritme_token')).toBe(false);
    });

    it('keeps the token on a non-JSON 401 (a proxy / Basic-auth gate)', async () => {
      const env = fakeBrowser(() =>
        Promise.resolve(new Response('<html>401</html>', { status: 401, headers: { 'Content-Type': 'text/html' } })),
      );
      const { apiClient, getApiErrorStatus, getApiErrorMessage } = await load();

      const error = await apiClient.get('/profile').catch((e: unknown) => e);
      expect(getApiErrorStatus(error)).toBe(401);
      expect(getApiErrorMessage(error)).toBeUndefined();
      expect(env.store.get('ritme_token')).toBe(TOKEN);
    });

    it('keeps the token on a 401 for a token the app no longer holds', async () => {
      const refresh: { setToken?: (t: string) => void } = {};
      const env = fakeBrowser(async () => {
        // A refresh landed while this request was in flight.
        refresh.setToken?.('new-token');
        return json(401, { error_code: 'token_revoked' });
      });
      const mod = await load();
      const session = await import('@/shared/session');
      refresh.setToken = session.setAuthToken;

      await mod.apiClient.get('/profile').catch(() => undefined);
      expect(env.store.get('ritme_token')).toBe('new-token');
    });

    it('keeps the token on a 401 whose error_code does not end sessions, and on 5xx', async () => {
      const replies = [json(401, { error_code: 'something_else' }), json(500, { message: 'boom' })];
      const env = fakeBrowser(() => Promise.resolve(replies.shift() as Response));
      const { apiClient, getApiErrorStatus, getApiErrorMessage } = await load();

      const e401 = await apiClient.get('/profile').catch((e: unknown) => e);
      const e500 = await apiClient.get('/profile').catch((e: unknown) => e);
      expect(getApiErrorStatus(e401)).toBe(401);
      expect(getApiErrorStatus(e500)).toBe(500);
      expect(getApiErrorMessage(e500)).toBe('boom');
      expect(env.store.get('ritme_token')).toBe(TOKEN);
    });

    it('rejects a network error with no response and keeps the token', async () => {
      const env = fakeBrowser(() => Promise.reject(new TypeError('Failed to fetch')));
      const { apiClient, ApiError, getApiErrorStatus } = await load();

      const error = await apiClient.get('/profile').catch((e: unknown) => e);
      expect(error).toBeInstanceOf(ApiError);
      expect((error as InstanceType<typeof ApiError>).code).toBe('network');
      expect((error as InstanceType<typeof ApiError>).response).toBeUndefined();
      expect(getApiErrorStatus(error)).toBeUndefined();
      expect(env.store.get('ritme_token')).toBe(TOKEN);
    });

    it('times out after 15 s', async () => {
      vi.useFakeTimers();
      fakeBrowser(
        (_url, init) =>
          new Promise((_resolve, reject) => {
            init.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
          }),
      );
      const { apiClient, REQUEST_TIMEOUT_MS } = await load();
      expect(REQUEST_TIMEOUT_MS).toBe(15_000);

      const pending = apiClient.get('/profile').catch((e: unknown) => e);
      await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS);
      const error = (await pending) as { code?: string; response?: unknown };
      expect(error.code).toBe('timeout');
      expect(error.response).toBeUndefined();
    });

    it('surfaces the server message of a validation error', async () => {
      fakeBrowser(() => Promise.resolve(json(422, { success: false, message: 'Invalid weight' })));
      const { apiClient, getApiErrorMessage, getApiErrorStatus } = await load();

      const error = await apiClient.post('/profile', {}).catch((e: unknown) => e);
      expect(getApiErrorStatus(error)).toBe(422);
      expect(getApiErrorMessage(error)).toBe('Invalid weight');
    });
  });
});
