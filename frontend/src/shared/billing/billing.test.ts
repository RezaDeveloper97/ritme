import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { getBillingChannel, storePurchase, storeQueryPurchases, StoreBillingError } from './billing';

type W = {
  RitmeNative?: Record<string, unknown>;
  __ritmeBilling?: { onResult: (r: unknown) => void };
};
const fake: W = {};
const w = () => fake;
const P = { signed_data: 'd', signature: 's', sku: 'k', token: 't' };

function install(channel = 'bazaar', version = 2) {
  const calls: Array<[string, string, string?]> = [];
  w().RitmeNative = {
    bridgeVersion: () => version,
    billingChannel: () => channel,
    purchase: (id: string, sku: string, payload: string) => calls.push([id, sku, payload]),
    queryPurchases: (id: string) => calls.push([id, 'query']),
  };
  return calls;
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal('window', fake);
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  delete w().RitmeNative;
  delete w().__ritmeBilling;
});

describe('getBillingChannel', () => {
  it('is web without a bridge, with an old bridge or without the method', () => {
    expect(getBillingChannel()).toBe('web');
    install('bazaar', 1);
    expect(getBillingChannel()).toBe('web');
    w().RitmeNative = { bridgeVersion: () => 2 };
    expect(getBillingChannel()).toBe('web');
  });
  it('reports the store from a v2 bridge', () => {
    install('myket');
    expect(getBillingChannel()).toBe('myket');
    install('other');
    expect(getBillingChannel()).toBe('web');
  });
});

describe('store requests', () => {
  it('dispatches results by requestId across concurrent requests', async () => {
    const calls = install();
    const a = storePurchase('sku_a', 'pa');
    const q = storeQueryPurchases();
    expect(calls.map((c) => c.slice(1))).toEqual([['sku_a', 'pa'], ['query']]);
    const [idA] = calls[0];
    const [idQ] = calls[1];
    w().__ritmeBilling!.onResult({ requestId: idQ, ok: true, purchases: [] });
    w().__ritmeBilling!.onResult({ requestId: 'unknown', ok: false });
    w().__ritmeBilling!.onResult({ requestId: idA, ok: true, purchases: [P] });
    await expect(a).resolves.toEqual([P]);
    await expect(q).resolves.toEqual([]);
  });

  it('rejects with the shell error code', async () => {
    const calls = install();
    const p = storePurchase('k', 'x');
    w().__ritmeBilling!.onResult({ requestId: calls[0][0], ok: false, error: 'cancelled' });
    await expect(p).rejects.toMatchObject({ reason: 'cancelled' });
  });

  it('times out (query 20 s, purchase 5 min) and ignores a late result', async () => {
    const calls = install();
    const q = storeQueryPurchases();
    const assertion = expect(q).rejects.toMatchObject({ reason: 'timeout' });
    await vi.advanceTimersByTimeAsync(20_000);
    await assertion;
    expect(() => w().__ritmeBilling!.onResult({ requestId: calls[0][0], ok: true })).not.toThrow();

    const p = storePurchase('k', 'x');
    const pa = expect(p).rejects.toBeInstanceOf(StoreBillingError);
    await vi.advanceTimersByTimeAsync(5 * 60_000);
    await pa;
  });

  it('is unavailable on the web', async () => {
    await expect(storePurchase('k', 'x')).rejects.toMatchObject({ reason: 'unavailable' });
  });
});
