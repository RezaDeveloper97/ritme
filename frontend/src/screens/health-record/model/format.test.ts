import { describe, expect, it } from 'vitest';

import type { PregnancyEntry } from '@/entities/health-record';

import { births, manualEntries, medicationSchedule, regularityKey } from './format';

const entry = (p: Partial<PregnancyEntry>): PregnancyEntry => ({
  id: null,
  source: 'tracked',
  outcome: 'ended',
  date: null,
  babyCount: null,
  editable: false,
  ...p,
});

describe('health record format helpers', () => {
  it('reads a medication schedule', () => {
    expect(medicationSchedule({ weekdays: [0, 1, 2, 3, 4, 5, 6] })).toEqual({ kind: 'daily' });
    expect(medicationSchedule({ weekdays: [0, 2, 4] })).toEqual({ kind: 'weekly', days: 3 });
    expect(medicationSchedule({ weekdays: [] })).toEqual({ kind: 'daily' });
  });

  it('maps regularity codes', () => {
    expect(regularityKey('regular')).toBe('regular');
    expect(regularityKey('whatever')).toBe('not_enough_data');
  });

  it('picks manual entries and births', () => {
    const items = [
      entry({ outcome: 'ongoing' }),
      entry({ id: 3, source: 'manual', outcome: 'cesarean', editable: true }),
      entry({ outcome: 'birth' }),
      entry({ outcome: 'ended' }),
    ];
    expect(manualEntries(items).map((e) => e.id)).toEqual([3]);
    expect(births(items).map((e) => e.outcome)).toEqual(['cesarean', 'birth']);
  });
});
