import { describe, expect, it, vi } from 'vitest';

import {
  createSessionRefresher,
  isRefreshDue,
  REFRESH_WINDOW_MS,
  RETRY_AFTER_MS,
  type RefresherDeps,
  type RefreshOutcome,
} from './refresh';

const DAY = 24 * 60 * 60 * 1000;
const NOW = Date.UTC(2027, 7, 25);

function setup(over: Partial<RefresherDeps> = {}, expiresIn = 10 * DAY) {
  let token: string | null = 'old';
  let now = NOW;
  let resolve!: (o: RefreshOutcome) => void;
  const request = vi.fn(
    () =>
      new Promise<RefreshOutcome>((r) => {
        resolve = r;
      }),
  );
  const setToken = vi.fn((t: string) => {
    token = t;
  });
  const refresher = createSessionRefresher({
    getToken: () => token,
    setToken,
    expiresAt: () => NOW + expiresIn,
    request,
    now: () => now,
    ...over,
  });
  return {
    refresher,
    request,
    setToken,
    answer: (o: RefreshOutcome) => resolve(o),
    setNow: (n: number) => (now = n),
    setCurrent: (t: string | null) => (token = t),
    current: () => token,
  };
}

describe('isRefreshDue', () => {
  it('is due only inside the window and before expiry', () => {
    expect(isRefreshDue(NOW + 365 * DAY, NOW)).toBe(false);
    expect(isRefreshDue(NOW + REFRESH_WINDOW_MS + 1, NOW)).toBe(false);
    expect(isRefreshDue(NOW + REFRESH_WINDOW_MS - 1, NOW)).toBe(true);
    expect(isRefreshDue(NOW + 1, NOW)).toBe(true);
    expect(isRefreshDue(NOW, NOW)).toBe(false);
    expect(isRefreshDue(NOW - DAY, NOW)).toBe(false);
    expect(isRefreshDue(null, NOW)).toBe(false);
  });
});

describe('createSessionRefresher', () => {
  it('makes no request when the expiry is far away', async () => {
    const t = setup({}, 200 * DAY);
    await t.refresher.refreshIfDue();
    expect(t.request).not.toHaveBeenCalled();
  });

  it('is single-flight: concurrent callers share one request', async () => {
    const t = setup();
    const calls = [t.refresher.refreshIfDue(), t.refresher.refreshIfDue(), t.refresher.refreshIfDue()];
    expect(t.request).toHaveBeenCalledTimes(1);
    expect(calls[1]).toBe(calls[0]);

    t.answer({ kind: 'refreshed', token: 'new' });
    await Promise.all(calls);
    expect(t.setToken).toHaveBeenCalledTimes(1);
    expect(t.current()).toBe('new');
  });

  it('does not store a refreshed token if the session changed meanwhile', async () => {
    const t = setup();
    const p = t.refresher.refreshIfDue();
    t.setCurrent(null); // signed out while in flight
    t.answer({ kind: 'refreshed', token: 'new' });
    await p;
    expect(t.setToken).not.toHaveBeenCalled();
  });

  it('keeps the token and never rejects when the request fails', async () => {
    const t = setup({ request: vi.fn(() => Promise.reject(new Error('offline'))) });
    await expect(t.refresher.refreshIfDue()).resolves.toBeUndefined();
    expect(t.current()).toBe('old');
  });

  it('waits before retrying a failed attempt', async () => {
    const request = vi.fn(() => Promise.reject(new Error('offline')));
    const t = setup({ request });
    await t.refresher.refreshIfDue();
    await t.refresher.refreshIfDue();
    expect(request).toHaveBeenCalledTimes(1);

    t.setNow(NOW + RETRY_AFTER_MS + 1);
    await t.refresher.refreshIfDue();
    expect(request).toHaveBeenCalledTimes(2);
  });

  it('stops asking a backend without the endpoint', async () => {
    const request = vi.fn(() => Promise.resolve<RefreshOutcome>({ kind: 'unavailable' }));
    const t = setup({ request });
    await t.refresher.refreshIfDue();
    t.setNow(NOW + 2 * RETRY_AFTER_MS);
    await t.refresher.refreshIfDue();
    expect(request).toHaveBeenCalledTimes(1);
  });

  it('does nothing without a token', async () => {
    const t = setup();
    t.setCurrent(null);
    await t.refresher.refreshIfDue();
    expect(t.request).not.toHaveBeenCalled();
  });
});
