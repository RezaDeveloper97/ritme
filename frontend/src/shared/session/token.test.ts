import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * token.ts against a minimal fake browser: a cookie jar behind
 * `document.cookie`, a Map behind `localStorage`, and a spy `fetch`.
 */
const TOKEN = 'eyJhbGciOi.eyJzdWIiOiIxIn0.c2ln';

function fakeBrowser() {
  const jar = new Map<string, string>();
  const store = new Map<string, string>();
  const writes: string[] = [];
  const fetchSpy = vi.fn<(input: string, init?: RequestInit) => Promise<Response>>(() =>
    Promise.resolve(new Response(null, { status: 204 })),
  );

  const document = {
    get cookie() {
      return [...jar].map(([k, v]) => `${k}=${v}`).join('; ');
    },
    set cookie(line: string) {
      writes.push(line);
      const [pair, ...attrs] = line.split(';').map((p) => p.trim());
      const [name, value] = pair.split('=');
      if (attrs.some((a) => a === 'max-age=0')) jar.delete(name);
      else jar.set(name, value);
    },
  };
  const window = {
    localStorage: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
      removeItem: (k: string) => void store.delete(k),
    },
    dispatchEvent: vi.fn(),
  };

  vi.stubGlobal('window', window);
  vi.stubGlobal('document', document);
  vi.stubGlobal('fetch', fetchSpy);
  return { jar, store, writes, fetchSpy };
}

async function loadToken() {
  vi.resetModules();
  return import('./token');
}

describe('token.ts flag handling', () => {
  let env: ReturnType<typeof fakeBrowser>;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-09-19T12:00:00Z'));
    env = fakeBrowser();
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('setAuthToken writes the flag synchronously and asks the server to re-issue it', async () => {
    const { setAuthToken, hasAuthCookie } = await loadToken();
    setAuthToken(TOKEN);

    expect(hasAuthCookie()).toBe(true);
    expect(env.store.get('ritme_token')).toBe(TOKEN);
    expect(env.fetchSpy).toHaveBeenCalledTimes(1);
    expect(env.fetchSpy.mock.calls[0][0]).toBe('/api/session/flag');
    expect((env.fetchSpy.mock.calls[0][1] as RequestInit).method).toBe('POST');
  });

  it('never puts the token in a cookie or in the flag request (CLAUDE.md §11)', async () => {
    const { setAuthToken, assertAuthFlag } = await loadToken();
    setAuthToken(TOKEN);
    vi.advanceTimersByTime(120_000);
    assertAuthFlag();

    for (const line of env.writes) expect(line).not.toContain(TOKEN);
    for (const call of env.fetchSpy.mock.calls) expect(JSON.stringify(call)).not.toContain(TOKEN);
  });

  it('assertAuthFlag restores a missing flag without touching the token', async () => {
    env.store.set('ritme_token', TOKEN);
    const { assertAuthFlag, hasAuthCookie, getAuthToken } = await loadToken();

    assertAuthFlag();
    expect(hasAuthCookie()).toBe(true);
    expect(getAuthToken()).toBe(TOKEN);
  });

  it('assertAuthFlag does not rewrite a present flag from script (keeps the server copy)', async () => {
    env.jar.set('ritme_auth', '1');
    const { assertAuthFlag } = await loadToken();

    assertAuthFlag();
    expect(env.writes).toHaveLength(0);
    expect(env.fetchSpy).toHaveBeenCalledTimes(1);
  });

  it('re-asserts with the server on start and resume, throttled to once a minute', async () => {
    env.jar.set('ritme_auth', '1');
    const { assertAuthFlag } = await loadToken();

    assertAuthFlag(); // start
    assertAuthFlag(); // resume right after
    expect(env.fetchSpy).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(61_000);
    assertAuthFlag(); // a later resume
    expect(env.fetchSpy).toHaveBeenCalledTimes(2);
  });

  it('survives an offline flag request', async () => {
    env.fetchSpy.mockImplementation(() => Promise.reject(new TypeError('Failed to fetch')));
    const { setAuthToken, hasAuthCookie, getAuthToken } = await loadToken();

    setAuthToken(TOKEN);
    await vi.runAllTimersAsync();
    expect(hasAuthCookie()).toBe(true);
    expect(getAuthToken()).toBe(TOKEN);
  });

  it('clearAuthToken drops token, flag and onboarding marker', async () => {
    env.jar.set('ritme_onboarding', 'name');
    const { setAuthToken, clearAuthToken, hasAuthCookie, getAuthToken } = await loadToken();
    setAuthToken(TOKEN);

    clearAuthToken();
    expect(getAuthToken()).toBeNull();
    expect(hasAuthCookie()).toBe(false);
    expect(env.jar.has('ritme_onboarding')).toBe(false);
  });
});
