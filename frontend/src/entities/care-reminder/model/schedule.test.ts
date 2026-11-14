import { describe, expect, it } from 'vitest';

import { doseDetail, frequencySummary, medicationSchedule, nextDoseAt, relativeDose } from './schedule';

describe('medicationSchedule', () => {
  it('words the weekdays as every day, every other day or a count', () => {
    expect(medicationSchedule([0, 2, 4, 6])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([1, 3, 5])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([5, 1, 3, 3])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([0, 2])).toEqual({ kind: 'daysPerWeek', count: 2 });
    expect(medicationSchedule([0, 3])).toEqual({ kind: 'daysPerWeek', count: 2 });
    expect(medicationSchedule([0, 1, 2, 3, 4, 5, 6])).toEqual({ kind: 'everyDay' });
    expect(medicationSchedule([])).toEqual({ kind: 'everyDay' });
  });
});

describe('frequencySummary', () => {
  it('words day / hour frequencies', () => {
    expect(frequencySummary({ every: 1, unit: 'day' })).toEqual({ kind: 'everyDay' });
    expect(frequencySummary({ every: 3, unit: 'day' })).toEqual({ kind: 'everyDays', count: 3 });
    expect(frequencySummary({ every: 2, unit: 'hour' })).toEqual({ kind: 'everyHours', count: 2 });
  });
});

describe('doseDetail', () => {
  it('is null without an amount or a usable unit', () => {
    expect(doseDetail({ amount: null, unit: null, unitLabel: null, meal: null })).toBeNull();
    expect(doseDetail({ amount: 1, unit: 'weird', unitLabel: null, meal: null })).toBeNull();
  });
  it('keeps a known unit and falls back to the server label', () => {
    expect(doseDetail({ amount: 1, unit: 'pill', unitLabel: 'قرص', meal: 'after' })).toMatchObject({ unit: 'pill', meal: 'after' });
    expect(doseDetail({ amount: 2, unit: 'weird', unitLabel: 'عدد', meal: null })).toMatchObject({ unit: null, unitLabel: 'عدد' });
  });
});

describe('nextDoseAt', () => {
  const base = { isActive: true, times: ['08:00', '20:00'], weekdays: [], frequency: null, startsOn: null, endsOn: null };
  const at = (y: number, mo: number, d: number, h = 0, mi = 0) => new Date(y, mo - 1, d, h, mi);

  it('picks the next slot today, then tomorrow morning', () => {
    expect(nextDoseAt(base, at(2026, 11, 11, 9, 0))).toEqual(at(2026, 11, 11, 20, 0));
    expect(nextDoseAt(base, at(2026, 11, 11, 21, 0))).toEqual(at(2026, 11, 12, 8, 0));
    expect(nextDoseAt(base, at(2026, 11, 11, 8, 0))).toEqual(at(2026, 11, 11, 8, 0));
  });
  it('is null when inactive, slotless or ended', () => {
    expect(nextDoseAt({ ...base, isActive: false }, at(2026, 11, 11))).toBeNull();
    expect(nextDoseAt({ ...base, times: [] }, at(2026, 11, 11))).toBeNull();
    expect(nextDoseAt({ ...base, endsOn: '2026-11-10' }, at(2026, 11, 11))).toBeNull();
  });
  it('waits for startsOn and stops after endsOn', () => {
    expect(nextDoseAt({ ...base, startsOn: '2026-11-14' }, at(2026, 11, 11, 9))).toEqual(at(2026, 11, 14, 8, 0));
    expect(nextDoseAt({ ...base, endsOn: '2026-11-11' }, at(2026, 11, 11, 21))).toBeNull();
  });
  it('honours every-N-days from startsOn', () => {
    const med = { ...base, frequency: { every: 3, unit: 'day' as const }, startsOn: '2026-11-10' };
    expect(nextDoseAt(med, at(2026, 11, 11, 9))).toEqual(at(2026, 11, 13, 8, 0));
    expect(nextDoseAt({ ...med, frequency: { every: 4, unit: 'hour' as const } }, at(2026, 11, 11, 9))).toEqual(
      at(2026, 11, 11, 20, 0),
    );
  });
  it('uses Saturday-based weekdays for old medications', () => {
    // 2026-11-11 is a Wednesday (= 4); only Saturday (0) -> 2026-11-14.
    expect(nextDoseAt({ ...base, weekdays: [0] }, at(2026, 11, 11, 9))).toEqual(at(2026, 11, 14, 8, 0));
    expect(nextDoseAt({ ...base, weekdays: [4] }, at(2026, 11, 11, 9))).toEqual(at(2026, 11, 11, 20, 0));
  });
});

describe('relativeDose', () => {
  const now = new Date(2026, 10, 11, 10, 0, 0);
  it('buckets the distance', () => {
    expect(relativeDose(new Date(2026, 10, 11, 10, 0, 30), now)).toEqual({ kind: 'now' });
    expect(relativeDose(new Date(2026, 10, 11, 10, 25), now)).toEqual({ kind: 'minutes', n: 25 });
    expect(relativeDose(new Date(2026, 10, 11, 12, 30), now)).toEqual({ kind: 'hours', n: 2 });
    expect(relativeDose(new Date(2026, 10, 13, 10, 0), now)).toEqual({ kind: 'days', n: 2 });
  });
});
