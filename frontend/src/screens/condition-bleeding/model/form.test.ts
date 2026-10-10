import { describe, expect, it } from 'vitest';

import type { PbacDay } from '@/entities/condition';

import { entryFrom, isDirty, liveScores, payloadFor } from './form';

const day: PbacDay = {
  date: '2026-09-22',
  light: 1,
  medium: 2,
  heavy: 3,
  clots: 'none',
  flooding: false,
  score: 71,
  period: { start: '2026-09-21', end: '2026-09-30', day: 2, score: 131, daysLogged: 2 },
  alertThreshold: 100,
  alert: null,
};

describe('liveScores', () => {
  it('swaps the saved day out of the period total', () => {
    expect(liveScores(day, entryFrom(day))).toEqual({ today: 71, period: 131 });
    expect(liveScores(day, { ...entryFrom(day), heavy: 0, flooding: true })).toEqual({ today: 16, period: 76 });
  });

  it('starts a period from today when there is none', () => {
    const empty = { ...day, light: 0, medium: 0, heavy: 0, clots: null, score: 0, period: null };
    expect(liveScores(empty, { ...entryFrom(empty), medium: 1 })).toEqual({ today: 5, period: 5 });
  });
});

describe('isDirty / payloadFor', () => {
  it('detects a change and sends clots only when it changed', () => {
    const saved = entryFrom(day);
    expect(isDirty(saved, saved)).toBe(false);
    const next = { ...saved, light: 2 };
    expect(isDirty(saved, next)).toBe(true);
    expect(payloadFor(saved, next)).toEqual({ light: 2, medium: 2, heavy: 3, flooding: false });
    expect(payloadFor(saved, { ...saved, clots: 'small' })).toMatchObject({ clots: 'small' });
  });
});
