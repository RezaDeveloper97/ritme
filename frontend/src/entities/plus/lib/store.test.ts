import { describe, expect, it } from 'vitest';

import type { PlusCatalog, PlusPlan } from '../model/types';
import { planDuration, plansForChannel, storeAvailable } from './store';

const plan = (id: number, over: Partial<PlusPlan> = {}): PlusPlan => ({
  id, code: `p${id}`, title: 't', badge: null, durationMonths: 1, price: 1, monthlyPrice: 1,
  savingsPercent: 0, isHighlighted: false, durationMinutes: null, aiCredits: 0, storeSkus: {}, ...over,
});

describe('store helpers', () => {
  it('keeps only plans with a SKU for the store', () => {
    const plans = [plan(1, { storeSkus: { bazaar: 'a' } }), plan(2, { storeSkus: { myket: 'b' } }), plan(3)];
    expect(plansForChannel(plans, 'bazaar').map((p) => p.id)).toEqual([1]);
    expect(plansForChannel(plans, 'myket').map((p) => p.id)).toEqual([2]);
  });
  it('checks the server-configured stores', () => {
    const c = { storeChannels: ['bazaar'] } as PlusCatalog;
    expect(storeAvailable(c, 'bazaar')).toBe(true);
    expect(storeAvailable(c, 'myket')).toBe(false);
  });
  it('prefers minutes over months', () => {
    expect(planDuration(plan(1, { durationMinutes: 5 }))).toEqual({ unit: 'minutes', value: 5 });
    expect(planDuration(plan(1, { durationMonths: 3 }))).toEqual({ unit: 'months', value: 3 });
  });
});
