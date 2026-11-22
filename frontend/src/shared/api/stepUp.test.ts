import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * Step-up (T-M12-06): the code request / verify calls and the X-Step-Up header
 * the export and delete calls carry, through the real fetch client.
 */
const TOKEN = 'eyJhbGciOi.eyJzdWIiOiIxIn0.c2ln';
const BASE = 'https://api.ritme.app/api/v1';

type FetchImpl = (input: string, init: RequestInit) => Promise<Response>;

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

function fakeBrowser(fetchImpl: FetchImpl) {
  const store = new Map<string, string>([['ritme_token', TOKEN]]);
  const fetchSpy = vi.fn<FetchImpl>((input, init) =>
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
  vi.stubGlobal('document', { cookie: '', documentElement: { lang: 'fa' } });
  vi.stubGlobal('fetch', fetchSpy);
  return { store, calls: () => fetchSpy.mock.calls.filter(([url]) => url.startsWith(BASE)) };
}

async function load() {
  vi.resetModules();
  return import('./index');
}

describe('step-up (T-M12-06)', () => {
  afterEach(() => vi.unstubAllGlobals());

  describe('with a server that answers', () => {
    let env: ReturnType<typeof fakeBrowser>;
    beforeEach(() => {
      env = fakeBrowser((url) =>
        Promise.resolve(
          url.endsWith('/auth/verify-otp')
            ? json(200, { success: true, data: { step_up_token: 'su-1', expires_in: 300 } })
            : json(200, { success: true, data: {} }),
        ),
      );
    });

    it('asks for a code with purpose step_up and no mobile (the API uses the token’s user)', async () => {
      const { requestStepUpCode } = await load();
      await requestStepUpCode();
      const [url, init] = env.calls()[0];
      expect(url).toBe(`${BASE}/auth/send-otp`);
      expect(JSON.parse(init.body as string)).toEqual({ purpose: 'step_up' });
      expect((init.headers as Record<string, string>).Authorization).toBe(`Bearer ${TOKEN}`);
    });

    it('exchanges the code for the step-up token and keeps the login token untouched', async () => {
      const { verifyStepUpCode } = await load();
      await expect(verifyStepUpCode('1234')).resolves.toBe('su-1');
      expect(JSON.parse(env.calls()[0][1].body as string)).toEqual({ purpose: 'step_up', code: '1234' });
      expect(env.store.get('ritme_token')).toBe(TOKEN);
    });

    it('sends X-Step-Up as a header, never letting extra headers replace the client’s own', async () => {
      const { apiClient, stepUpHeaders } = await load();
      await apiClient.delete('/account', {
        headers: { ...stepUpHeaders('su-1'), Authorization: 'Bearer forged', 'accept-language': 'xx' },
      });
      const [url, init] = env.calls()[0];
      const headers = init.headers as Record<string, string>;
      expect(url).toBe(`${BASE}/account`);
      expect(headers['X-Step-Up']).toBe('su-1');
      expect(headers.Authorization).toBe(`Bearer ${TOKEN}`);
      expect(headers['Accept-Language']).toBe('fa');
      expect(headers['accept-language']).toBeUndefined();
    });
  });

  it('rejects a verify answer without a token (a login answer is never taken for a step-up)', async () => {
    fakeBrowser(() => Promise.resolve(json(200, { success: true, data: { access_token: 'login' } })));
    const { verifyStepUpCode } = await load();
    await expect(verifyStepUpCode('1234')).rejects.toThrow('step-up');
  });

  it('recognises the 403 step_up_required — and only that — without ending the session', async () => {
    const env = fakeBrowser((url) =>
      Promise.resolve(
        url.endsWith('/profile/export')
          ? json(403, { success: false, message: 'x', error_code: 'step_up_required' })
          : json(403, { success: false, message: 'x', error_code: 'ai_feature_disabled' }),
      ),
    );
    const { apiClient, isStepUpRequired } = await load();
    const refused = await apiClient.get('/profile/export').catch((e: unknown) => e);
    const other = await apiClient.get('/other').catch((e: unknown) => e);
    expect(isStepUpRequired(refused)).toBe(true);
    expect(isStepUpRequired(other)).toBe(false);
    expect(isStepUpRequired(new Error('x'))).toBe(false);
    expect(env.store.get('ritme_token')).toBe(TOKEN);
  });
});
