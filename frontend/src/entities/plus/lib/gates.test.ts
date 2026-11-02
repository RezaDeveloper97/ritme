import { describe, expect, it } from 'vitest';

import type { PlusEntitlement, PlusStatus } from '../model/types';
import { detailLocked, isAssistantFairUse, isLongRange, isRangeDenial, labQuotaOf, rangeLocked } from './gates';

const ent = (key: string, o: Partial<PlusEntitlement> = {}): PlusEntitlement => ({
  key, allowed: true, unlimited: false, limit: null, used: 0, remaining: null, ...o,
});
const status = (entitlements: PlusEntitlement[]) => ({ entitlements }) as unknown as PlusStatus;

describe('range gate', () => {
  it('only 1y and all are long', () => {
    expect(['7d', '2w', '1m', '3m', '6m'].some(isLongRange)).toBe(false);
    expect(isLongRange('1y') && isLongRange('all')).toBe(true);
  });
  it('locks long ranges for free only', () => {
    const free = status([ent('plus.analysis_long_range', { allowed: false, limit: 0 })]);
    const plus = status([ent('plus.analysis_long_range', { unlimited: true })]);
    expect(rangeLocked(free, '1y')).toBe(true);
    expect(rangeLocked(free, 'all')).toBe(true);
    expect(rangeLocked(free, '6m')).toBe(false);
    expect(rangeLocked(plus, '1y')).toBe(false);
    expect(rangeLocked(undefined, '1y')).toBe(false);
    expect(rangeLocked(status([]), '1y')).toBe(false);
  });
});

describe('detail lock', () => {
  it('follows plus.deep_analysis', () => {
    expect(detailLocked(status([ent('plus.deep_analysis', { allowed: false })]))).toBe(true);
    expect(detailLocked(status([ent('plus.deep_analysis')]))).toBe(false);
    expect(detailLocked(undefined)).toBe(false);
  });
});

describe('lab quota', () => {
  it('reads limit and remaining from the entitlement', () => {
    expect(labQuotaOf(null)).toBeNull();
    expect(labQuotaOf(ent('plus.lab_ai', { limit: 1, used: 0, remaining: 1 }))).toEqual({ state: 'available', limit: 1, remaining: 1 });
    expect(labQuotaOf(ent('plus.lab_ai', { limit: 1, used: 1, remaining: 0 }))).toEqual({ state: 'spent', limit: 1 });
    expect(labQuotaOf(ent('plus.lab_ai', { limit: 10, used: 3, remaining: 7 }))).toEqual({ state: 'available', limit: 10, remaining: 7 });
    expect(labQuotaOf(ent('plus.lab_ai', { allowed: false }))).toEqual({ state: 'locked' });
    expect(labQuotaOf(ent('plus.lab_ai', { unlimited: true }))).toEqual({ state: 'unlimited' });
  });
});

describe('assistant fair use', () => {
  it('maps only a Plus exhaustion of the assistant', () => {
    expect(isAssistantFairUse({ kind: 'exhausted', feature: 'plus.assistant_unlimited' })).toBe(true);
    expect(isAssistantFairUse({ kind: 'quota', feature: 'plus.assistant_unlimited' })).toBe(false);
    expect(isAssistantFairUse({ kind: 'exhausted', feature: 'plus.lab_ai' })).toBe(false);
    expect(isAssistantFairUse(null)).toBe(false);
  });
});

describe('range denial', () => {
  it('matches a locked long-range 402 only', () => {
    expect(isRangeDenial({ kind: 'locked', feature: 'plus.analysis_long_range' })).toBe(true);
    expect(isRangeDenial({ kind: 'locked', feature: 'plus.lab_ai' })).toBe(false);
    expect(isRangeDenial(null)).toBe(false);
  });
});
