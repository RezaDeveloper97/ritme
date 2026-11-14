import { describe, expect, it } from 'vitest';

import type { CycleDayMarker } from '@/entities/cycle';

import { ringNextMilestone } from './ring-next';

// 28-day cycle: period 0–4, grey 5–9, fertile 10–15 with ovulation on 14, grey 16–22, PMS 23–27.
const M: (CycleDayMarker | null)[] = Array.from({ length: 28 }, (_, i) =>
  i <= 4 ? 'period'
  : i === 14 ? 'ovulation'
  : i >= 10 && i <= 15 ? 'fertile'
  : i >= 23 ? 'pms'
  : null);

describe('ringNextMilestone', () => {
  it('period day keeps the period read-out', () => {
    expect(ringNextMilestone(M, 2)).toBeNull();
  });
  it('grey day after the period counts to the fertile window', () => {
    expect(ringNextMilestone(M, 5)).toEqual({ kind: 'fertile', days: 5 });
    expect(ringNextMilestone(M, 9)).toEqual({ kind: 'fertile', days: 1 });
  });
  it('fertile day counts to ovulation', () => {
    expect(ringNextMilestone(M, 11)).toEqual({ kind: 'ovulation', days: 3 });
  });
  it('ovulation day is itself', () => {
    expect(ringNextMilestone(M, 14)).toEqual({ kind: 'ovulation', days: 0 });
  });
  it('fertile tail after ovulation counts to PMS', () => {
    expect(ringNextMilestone(M, 15)).toEqual({ kind: 'pms', days: 8 });
  });
  it('grey day after ovulation counts to PMS', () => {
    expect(ringNextMilestone(M, 18)).toEqual({ kind: 'pms', days: 5 });
  });
  it('PMS day falls back to the period countdown', () => {
    expect(ringNextMilestone(M, 25)).toBeNull();
  });
  it('grey day with only the period ahead falls back', () => {
    const noPms = M.map(m => (m === 'pms' ? null : m));
    expect(ringNextMilestone(noPms, 20)).toBeNull();
  });
  it('no fertile marks (teen) skips straight to what is left', () => {
    const teen = M.map(m => (m === 'fertile' || m === 'ovulation' ? null : m));
    expect(ringNextMilestone(teen, 6)).toEqual({ kind: 'pms', days: 17 });
  });
  it('unplaced day', () => {
    expect(ringNextMilestone(M, null)).toBeNull();
  });
});
