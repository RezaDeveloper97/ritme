import { describe, expect, it } from 'vitest';

import { isPillMethod, methodToPayload, packCellState, packWeeks } from './pack';
import type { ContraceptionMethod, PackDay } from './types';

const day = (n: number, over: Partial<PackDay> = {}): PackDay => ({
  day: n,
  date: `2026-10-${String(n).padStart(2, '0')}`,
  kind: 'active',
  status: 'upcoming',
  ...over,
});

const pill: ContraceptionMethod = {
  method: 'combined_pill',
  packType: '21_7',
  packStartedOn: '2026-09-16',
  packsLeft: 2,
  reminder: { enabled: true, time: '21:00' },
  insertedOn: null,
  iudLifetimeYears: null,
  followupOn: null,
  followupDone: false,
  iudReplaceOn: null,
  injectedOn: null,
  nextInjectionOn: null,
  replaceOn: null,
};

describe('packCellState', () => {
  it('marks today, taken and pending today', () => {
    expect(packCellState(day(8, { status: 'pending' }), '2026-10-08')).toBe('today');
    expect(packCellState(day(8, { status: 'taken' }), '2026-10-08')).toBe('todayTaken');
    expect(packCellState(day(3, { status: 'taken' }), '2026-10-08')).toBe('taken');
    expect(packCellState(day(4, { status: 'missed' }), '2026-10-08')).toBe('missed');
  });

  it('keeps break, placebo and pre-setup days apart', () => {
    expect(packCellState(day(22, { kind: 'break', status: null }), '2026-10-08')).toBe('break');
    expect(packCellState(day(26, { kind: 'placebo', status: 'upcoming' }), '2026-10-08')).toBe('placebo');
    expect(packCellState(day(2, { status: 'untracked' }), '2026-10-08')).toBe('untracked');
    expect(packCellState(day(9), '2026-10-08')).toBe('upcoming');
  });
});

describe('packWeeks', () => {
  it('chunks 28 days into four ordered weeks', () => {
    const weeks = packWeeks(Array.from({ length: 28 }, (_, i) => day(28 - i)));
    expect(weeks).toHaveLength(4);
    expect(weeks[0]!.map((d) => d.day)).toEqual([1, 2, 3, 4, 5, 6, 7]);
    expect(weeks[3]![6]!.day).toBe(28);
  });
});

describe('methodToPayload', () => {
  it('re-sends a pill method with its reminder', () => {
    expect(methodToPayload(pill)).toEqual({
      method: 'combined_pill',
      pack_type: '21_7',
      pack_started_on: '2026-09-16',
      packs_left: 2,
      reminder_time: '21:00',
      reminder_enabled: true,
    });
  });

  it('drops the pack type of the progestogen-only pill', () => {
    expect(methodToPayload({ ...pill, method: 'progestin_pill' }).pack_type).toBeNull();
  });

  it('sends only the fields of a long-acting method', () => {
    expect(
      methodToPayload({ ...pill, method: 'injection', packType: null, reminder: null, injectedOn: '2026-08-04' }),
    ).toEqual({ method: 'injection', injected_on: '2026-08-04' });
    expect(methodToPayload({ ...pill, method: 'condom' })).toEqual({ method: 'condom' });
  });
});

describe('isPillMethod', () => {
  it('knows the two pills', () => {
    expect(isPillMethod('combined_pill')).toBe(true);
    expect(isPillMethod('progestin_pill')).toBe(true);
    expect(isPillMethod('hormonal_iud')).toBe(false);
    expect(isPillMethod(null)).toBe(false);
  });
});
