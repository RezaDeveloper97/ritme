import { describe, expect, it } from 'vitest';

import type { PlusPlan, PlusStatus } from '../model/types';
import { defaultPlan, membershipOf, parseGatewayReturn, planFromParam, remainingShare, upgradePlan } from './plans';

const plan = (id: number, months: number, highlighted = false): PlusPlan => ({
  id,
  code: `plus_${months}m`,
  title: `${months}`,
  badge: null,
  durationMonths: months,
  price: months * 1000,
  monthlyPrice: 1000,
  savingsPercent: 0,
  isHighlighted: highlighted,
});
const plans = [plan(1, 1), plan(2, 3, true), plan(3, 6)];

describe('plan selection', () => {
  it('pre-selects the highlighted plan, else the first', () => {
    expect(defaultPlan(plans)?.id).toBe(2);
    expect(defaultPlan([plan(1, 1), plan(3, 6)])?.id).toBe(1);
    expect(defaultPlan([])).toBeNull();
  });

  it('honours ?plan= only for a known id', () => {
    expect(planFromParam(plans, '3')?.id).toBe(3);
    expect(planFromParam(plans, '99')?.id).toBe(2);
    expect(planFromParam(plans, null)?.id).toBe(2);
    expect(planFromParam(plans, 'x')?.id).toBe(2);
  });

  it('offers the longest longer plan as the upgrade', () => {
    expect(upgradePlan(plans, 3)?.id).toBe(3);
    expect(upgradePlan(plans, 1)?.id).toBe(3);
    expect(upgradePlan(plans, 6)).toBeNull();
    expect(upgradePlan(plans, null)).toBeNull();
  });
});

describe('remainingShare', () => {
  it('is days left over the period length, clamped', () => {
    expect(remainingShare(46, '2026-10-01T00:00:00+03:30', '2026-11-02T00:00:00+03:30')).toBe(1);
    expect(remainingShare(16, '2026-10-01T00:00:00+03:30', '2026-11-02T00:00:00+03:30')).toBe(0.5);
    expect(remainingShare(5, 'bad', 'bad')).toBe(0);
  });
});

describe('membershipOf', () => {
  const base: PlusStatus = {
    tier: 'free',
    isPlus: false,
    subscription: null,
    trial: null,
    trialAvailable: true,
    periodStart: '',
    resetsAt: '',
    entitlements: [],
  };
  it('prefers a running subscription, then a running trial', () => {
    expect(membershipOf(base).kind).toBe('none');
    const trial = { ...base, trial: { startedAt: '', endsAt: '', isActive: true, daysLeft: 3 } };
    expect(membershipOf(trial).kind).toBe('trial');
    const sub = {
      ...trial,
      subscription: {
        id: 1,
        plan: null,
        status: 'active',
        source: '',
        startsAt: '',
        endsAt: '',
        daysLeft: 10,
        autoRenew: true,
        canceledAt: null,
      },
    };
    expect(membershipOf(sub).kind).toBe('subscription');
  });
});

describe('parseGatewayReturn', () => {
  const q = (s: string) => new URLSearchParams(s);
  it('reads the return hop query', () => {
    expect(parseGatewayReturn(q('reference=ab&authority=FAKE-ab&status=OK'))).toEqual({
      reference: 'ab',
      authority: 'FAKE-ab',
      status: 'OK',
    });
    expect(parseGatewayReturn(q('reference=ab&authority=FAKE-ab&status=cancel'))?.status).toBe('NOK');
  });
  it('is null without a reference or authority', () => {
    expect(parseGatewayReturn(q('status=OK'))).toBeNull();
    expect(parseGatewayReturn(q('reference=ab'))).toBeNull();
  });
});
