import { describe, expect, it } from 'vitest';

import type { TreatmentItem, TreatmentTip } from '@/entities/menopause';

import {
  dotsSummary,
  earliestStart,
  groupWeekDots,
  inLocaleOrder,
  lifestyleSuggestions,
  lifestyleTipOf,
  tipBody,
  weekDates,
} from './treatment';

const DATES = ['2026-10-03', '2026-10-04', '2026-10-05', '2026-10-06', '2026-10-07', '2026-10-08', '2026-10-09'];

function item(id: number, patch: Partial<TreatmentItem> & { taken?: string[] }): TreatmentItem {
  const { taken = [], ...rest } = patch;
  return {
    id,
    kind: 'hrt',
    name: `item ${id}`,
    dose: null,
    schedule: 'morning',
    form: null,
    startedOn: null,
    reviewOn: null,
    stoppedOn: null,
    active: true,
    takenToday: false,
    week: DATES.map((date) => ({ date, taken: taken.includes(date), amount: null })),
    daysTaken: taken.length,
    days: 4,
    adherencePct: null,
    goal: null,
    reminder: null,
    ...rest,
  };
}

const tip = (code: string, title: string, extra: Partial<TreatmentTip> = {}): TreatmentTip => ({
  code,
  title,
  body: null,
  placement: 'treatment_lifestyle',
  weeklyGoal: 2,
  goalUnit: 'sessions',
  ...extra,
});

describe('treatment screen helpers (CB-MENO-10)', () => {
  it('marks a day done only when every running daily item was taken', () => {
    const a = item(1, { taken: ['2026-10-03', '2026-10-04', '2026-10-06'] });
    const b = item(2, { taken: ['2026-10-03', '2026-10-06'], startedOn: '2026-10-05' });
    const weekly = item(3, { schedule: 'weekly' });
    const dots = groupWeekDots([a, b, weekly], DATES, '2026-10-06');
    // Sat both, Sun b not started yet → a alone, Mon none, Tue both, then future.
    expect(dots).toEqual(['done', 'done', 'missed', 'done', 'future', 'future', 'future']);
    expect(dotsSummary(dots)).toEqual({ done: 3, days: 4 });
  });

  it('draws the days before an item existed as not scheduled, not missed (CB-MENO-13b)', () => {
    // added on Wednesday without a start date: the API marks Sat … Tue `scheduled: false`
    const gel = item(1, {
      week: DATES.map((date) => ({ date, taken: date === '2026-10-07', amount: null, scheduled: date >= '2026-10-07' })),
    });
    const dots = groupWeekDots([gel], DATES, '2026-10-07');
    expect(dots).toEqual(['off', 'off', 'off', 'off', 'done', 'future', 'future']);
    expect(dotsSummary(dots)).toEqual({ done: 1, days: 1 });
    // a day only one of two items was due on counts that item alone
    const old = item(2, { taken: ['2026-10-06'] });
    expect(groupWeekDots([gel, old], DATES, '2026-10-07').slice(3, 5)).toEqual(['done', 'missed']);
    expect(groupWeekDots([], DATES, '2026-10-07')[0]).toBe('off');
  });

  it('reorders the Saturday week for a Sunday-first calendar', () => {
    expect(inLocaleOrder(DATES, 'fa')[0]).toBe('2026-10-03');
    expect(inLocaleOrder(DATES, 'en')[0]).toBe('2026-10-04');
    expect(inLocaleOrder(DATES, 'en')).toHaveLength(7);
  });

  it('derives the week dates without items', () => {
    expect(weekDates('2026-10-03', [])).toEqual(DATES);
    expect(weekDates('2026-10-03', [item(1, {})])).toEqual(DATES);
  });

  it('finds the HRT start, catalog lifestyle tips and suggestions', () => {
    expect(earliestStart([item(1, { startedOn: '2026-08-01' }), item(2, { startedOn: '2026-07-20' }), item(3, {})])).toBe(
      '2026-07-20',
    );
    expect(earliestStart([item(1, {})])).toBeNull();
    const tips = [tip('lifestyle_resistance', 'ورزش مقاومتی'), tip('lifestyle_brisk_walk', 'پیاده‌روی تند')];
    const walk = item(4, { kind: 'lifestyle', name: ' پیاده‌روی تند ' });
    expect(lifestyleTipOf(walk, tips)?.code).toBe('lifestyle_brisk_walk');
    expect(lifestyleSuggestions([walk], tips).map((t) => t.code)).toEqual(['lifestyle_resistance']);
    expect(tipBody([tip('doctor_only', 'x', { body: 'only with your doctor', placement: 'treatment' })], 'doctor_only')).toBe(
      'only with your doctor',
    );
    expect(tipBody([], 'doctor_only')).toBeNull();
  });
});
