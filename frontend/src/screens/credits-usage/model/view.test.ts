import { describe, expect, it } from 'vitest';

import type { AiCredits } from '@/entities/ai-credit';

import { costRows, dayRows, extraKept, featureRows, usageStateOf } from './view';

const base: AiCredits = {
  canUse: true,
  period: 'plus',
  periodEndsAt: '2026-12-12T10:00:00+03:30',
  plan: { total: 100, used: 12, remaining: 88 },
  extra: { remaining: 50, locked: false },
  spendable: 138,
  features: [
    { key: 'lab', title: 'a', creditCost: 5, available: true, affordable: true },
    { key: 'assistant', title: 'b', creditCost: 1, available: true, affordable: true },
    { key: 'off', title: 'c', creditCost: 2, available: false, affordable: true },
  ],
};

describe('credits usage view', () => {
  it('names the three hero states', () => {
    expect(usageStateOf(base)).toBe('ready');
    expect(usageStateOf({ ...base, spendable: 0 })).toBe('exhausted');
    expect(usageStateOf({ ...base, canUse: false, spendable: 0 })).toBe('no_subscription');
  });
  it('flags kept extra credits only when locked and non-empty', () => {
    expect(extraKept(base)).toBe(false);
    expect(extraKept({ ...base, extra: { remaining: 50, locked: true } })).toBe(true);
    expect(extraKept({ ...base, extra: { remaining: 0, locked: true } })).toBe(false);
  });
  it('lists costs cheapest first without switched-off features', () => {
    expect(costRows(base).map((f) => f.key)).toEqual(['assistant', 'lab']);
  });
  it('sorts usage rows', () => {
    expect(featureRows([{ feature: 'a', title: '', uses: 1, credits: 1 }, { feature: 'b', title: '', uses: 1, credits: 4 }, { feature: 'c', title: '', uses: 0, credits: 0 }]).map((r) => r.feature)).toEqual(['b', 'a']);
    expect(dayRows([{ date: '2026-11-10', credits: 1 }, { date: '2026-11-02', credits: 2 }]).map((r) => r.date)).toEqual(['2026-11-02', '2026-11-10']);
  });
});
