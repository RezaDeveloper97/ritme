import { describe, expect, it } from 'vitest';

import { futureCycleDayMap, isLateDay, monthDayPaint, openPeriodEnd } from './monthPaint';
import type { CycleAnchors, CycleFutureCycle, CycleMonthDay } from './types';

function day(over: Partial<CycleMonthDay> = {}): CycleMonthDay {
  return {
    date: '2026-09-10',
    mainPhase: 'luteal',
    fertilityLevel: 'low',
    cycleDay: 17,
    estimatedOvulationDay: 15,
    phase: 'luteal',
    isPredicted: true,
    isPeriod: false,
    periodKind: null,
    daysToPeriod: 12,
    daysLate: 0,
    ...over,
  };
}

const cycle = (k: number, start: string, end: string): CycleFutureCycle => ({
  k,
  periodStart: start,
  periodEnd: end,
  fertileStart: null,
  fertileEnd: null,
  ovulation: null,
});

describe('monthDayPaint', () => {
  const none = new Map();

  it('paints logged, assumed and predicted period kinds from the server', () => {
    for (const kind of ['logged', 'assumed', 'predicted'] as const) {
      expect(monthDayPaint(day({ periodKind: kind, isPeriod: true }), 'x', none)).toEqual({
        marker: 'period',
        periodKind: kind,
        late: false,
      });
    }
  });

  it('marks 10 days late as late days, never as a period', () => {
    const late = day({ mainPhase: 'period_expected', daysLate: 10, daysToPeriod: 0 });
    expect(isLateDay(late)).toBe(true);
    expect(monthDayPaint(late, 'x', none)).toEqual({ marker: null, periodKind: null, late: true });
  });

  it('treats the predicted start day (not late yet) as a predicted period', () => {
    const start = day({ mainPhase: 'period_expected', daysLate: 0 });
    expect(isLateDay(start)).toBe(false);
    expect(monthDayPaint(start, 'x', none)?.periodKind).toBe('predicted');
  });

  it('paints fertile days and the ovulation day, and PMS in the last 3 days', () => {
    expect(monthDayPaint(day({ mainPhase: 'fertile', cycleDay: 12 }), 'x', none)?.marker).toBe('fertile');
    expect(monthDayPaint(day({ mainPhase: 'fertile', cycleDay: 15 }), 'x', none)?.marker).toBe('ovulation');
    expect(monthDayPaint(day({ daysToPeriod: 3 }), 'x', none)?.marker).toBe('pms');
    expect(monthDayPaint(day({ daysToPeriod: 4 }), 'x', none)).toBeNull();
  });

  it('paints no fertile marks when fertility is suppressed (pill)', () => {
    expect(monthDayPaint(day({ mainPhase: 'fertile', cycleDay: 12 }), 'x', none, true)).toBeNull();
    expect(monthDayPaint(day({ mainPhase: 'fertile', cycleDay: 15 }), 'x', none, true)).toBeNull();
  });

  it('paints nothing for an unplaced day (15 days late: no predicted days)', () => {
    const unknown = day({ mainPhase: 'unknown', cycleDay: null, daysToPeriod: null, daysLate: null });
    expect(monthDayPaint(unknown, '2026-10-01', futureCycleDayMap([]))).toBeNull();
  });

  it('falls back to the future_cycles map only for a day the payload lacks', () => {
    const map = futureCycleDayMap([cycle(1, '2026-10-19', '2026-10-21')]);
    expect(monthDayPaint(undefined, '2026-10-20', map)?.periodKind).toBe('predicted');
    expect(monthDayPaint(undefined, '2026-10-25', map)).toBeNull();
  });
});

describe('futureCycleDayMap', () => {
  it('paints the predicted period, 3 PMS days before it, the window and ovulation', () => {
    const map = futureCycleDayMap([
      {
        k: 1,
        periodStart: '2026-10-19',
        periodEnd: '2026-10-23',
        fertileStart: '2026-10-28',
        fertileEnd: '2026-11-02',
        ovulation: '2026-11-02',
      },
    ]);
    expect(map.get('2026-10-19')?.periodKind).toBe('predicted');
    expect(map.get('2026-10-23')?.periodKind).toBe('predicted');
    expect(map.get('2026-10-24')).toBeUndefined();
    expect(map.get('2026-10-16')?.marker).toBe('pms');
    expect(map.get('2026-10-18')?.marker).toBe('pms');
    expect(map.get('2026-10-15')).toBeUndefined();
    expect(map.get('2026-10-30')?.marker).toBe('fertile');
    expect(map.get('2026-11-02')?.marker).toBe('ovulation');
  });

  it('stops at k = 6: nothing beyond the sixth cycle', () => {
    const cycles = Array.from({ length: 8 }, (_, i) =>
      cycle(i + 1, `2027-0${i + 1}-10`, `2027-0${i + 1}-12`),
    );
    const map = futureCycleDayMap(cycles);
    expect(map.get('2027-06-10')?.periodKind).toBe('predicted');
    expect(map.get('2027-07-10')).toBeUndefined();
    expect(map.get('2027-08-11')).toBeUndefined();
  });

  it('drops fertile days and ovulation when suppressed', () => {
    const map = futureCycleDayMap(
      [{ ...cycle(1, '2026-10-19', '2026-10-21'), fertileStart: '2026-10-28', fertileEnd: '2026-11-02', ovulation: '2026-11-02' }],
      true,
    );
    expect(map.get('2026-10-30')).toBeUndefined();
    expect(map.get('2026-11-02')).toBeUndefined();
    expect(map.get('2026-10-19')?.periodKind).toBe('predicted');
  });

  it('is empty with no future cycles (15 days late, pregnancy)', () => {
    expect(futureCycleDayMap([]).size).toBe(0);
  });
});

describe('openPeriodEnd', () => {
  const anchors = (over: Partial<CycleAnchors> = {}): CycleAnchors => ({
    currentPeriodStart: '2026-10-01',
    currentPeriodStartSource: 'user_logged',
    currentPeriodEnd: '2026-10-05',
    currentPeriodEndSource: 'assumed',
    currentPeriodEndIsConfirmed: false,
    predictedNextPeriodStart: null,
    estimatedOvulationDate: null,
    ...over,
  });

  it('reads the engine end and whether it is confirmed', () => {
    expect(openPeriodEnd(anchors(), '2026-10-01')).toEqual({ end: '2026-10-05', confirmed: false });
    expect(openPeriodEnd(anchors({ currentPeriodEndIsConfirmed: true }), '2026-10-01')?.confirmed).toBe(true);
  });

  it('is null for another period or without an end', () => {
    expect(openPeriodEnd(anchors(), '2026-09-03')).toBeNull();
    expect(openPeriodEnd(anchors({ currentPeriodEnd: null }), '2026-10-01')).toBeNull();
    expect(openPeriodEnd(null, '2026-10-01')).toBeNull();
  });
});
