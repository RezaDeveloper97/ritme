import { describe, expect, it } from 'vitest';

import type { AiCredits } from '../model/types';
import { gateStateOf, planShare } from './status';

const credits = (over: Partial<AiCredits> = {}): AiCredits => ({
  canUse: true,
  period: 'plus',
  periodEndsAt: '2026-12-12T10:00:00+03:30',
  plan: { total: 100, used: 12, remaining: 88 },
  extra: { remaining: 0, locked: false },
  spendable: 88,
  features: [
    { key: 'assistant', title: 'a', creditCost: 1, available: true, affordable: true },
    { key: 'lab_analysis', title: 'l', creditCost: 5, available: true, affordable: true },
    { key: 'voice_log', title: 'v', creditCost: 2, available: false, affordable: false },
  ],
  ...over,
});

describe('gateStateOf', () => {
  it('fails open while the balance is unknown', () => {
    expect(gateStateOf(undefined, 'assistant', true)).toMatchObject({ status: 'loading', canUse: true });
    expect(gateStateOf(undefined, 'assistant', false)).toMatchObject({ status: 'ready', canUse: true, cost: null, balance: null });
  });
  it('is ready with enough credits and reports the cost', () => {
    expect(gateStateOf(credits(), 'lab_analysis', false)).toEqual({ status: 'ready', cost: 5, balance: 88, canUse: true });
  });
  it('needs a subscription when there is none (extra credits do not help)', () => {
    expect(gateStateOf(credits({ canUse: false, spendable: 0, extra: { remaining: 50, locked: true } }), 'assistant', false)).toMatchObject({
      status: 'no_subscription',
      canUse: false,
    });
  });
  it('runs out of credits by the server flag or by the cost', () => {
    expect(gateStateOf(credits({ spendable: 0 }), 'assistant', false).status).toBe('no_credits');
    expect(gateStateOf(credits({ spendable: 3 }), 'lab_analysis', false).status).toBe('no_credits');
    expect(gateStateOf(credits(), 'unknown_feature', false).status).toBe('ready');
  });
  it('reports a switched-off feature as disabled', () => {
    expect(gateStateOf(credits(), 'voice_log', false)).toMatchObject({ status: 'disabled', canUse: false });
  });
});

describe('planShare', () => {
  it('is the remaining share clamped to 0..1', () => {
    expect(planShare(credits())).toBeCloseTo(0.88);
    expect(planShare({ plan: { total: 0, used: 0, remaining: 0 } })).toBe(0);
    expect(planShare({ plan: { total: 10, used: 0, remaining: 20 } })).toBe(1);
  });
});
