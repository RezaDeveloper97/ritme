import { describe, expect, it } from 'vitest';

import {
  defaultDuration,
  durationOptions,
  emptyForm,
  mapServerErrors,
  returnHref,
  setTimesCount,
  slotLabel,
  slotPeriod,
  toggleWeekday,
  validateForm,
  weekdayOrder,
  weekdaySummary,
} from './form';

describe('setTimesCount (times ↔ count)', () => {
  it('grows with the defaults in order 08:00, 20:00, 14:00, 23:00', () => {
    expect(setTimesCount(['08:00'], 2)).toEqual(['08:00', '20:00']);
    expect(setTimesCount(['08:00'], 3)).toEqual(['08:00', '20:00', '14:00']);
    expect(setTimesCount(['08:00'], 4)).toEqual(['08:00', '20:00', '14:00', '23:00']);
  });

  it('keeps edited slots and skips defaults already taken', () => {
    expect(setTimesCount(['20:00'], 2)).toEqual(['20:00', '08:00']);
    expect(setTimesCount(['09:30', '21:00'], 3)).toEqual(['09:30', '21:00', '08:00']);
  });

  it('shrinks by dropping the last rows', () => {
    expect(setTimesCount(['08:00', '20:00', '14:00'], 1)).toEqual(['08:00']);
  });

  it('clamps the count to 1–4', () => {
    expect(setTimesCount(['08:00'], 0)).toHaveLength(1);
    expect(setTimesCount(['08:00'], 9)).toHaveLength(4);
  });

  it('every count yields exactly that many unique rows', () => {
    for (const n of [1, 2, 3, 4]) {
      const times = setTimesCount(['08:00'], n);
      expect(times).toHaveLength(n);
      expect(new Set(times).size).toBe(n);
    }
  });
});

describe('slotLabel', () => {
  it('drops the leading zero and keeps 24-hour time', () => {
    expect(slotLabel('08:00')).toBe('8:00');
    expect(slotLabel('20:05')).toBe('20:05');
    expect(slotLabel('00:30')).toBe('0:30');
  });
});

describe('slotPeriod', () => {
  it('buckets the day like the reminders list', () => {
    expect(slotPeriod('08:00')).toBe('morning');
    expect(slotPeriod('14:00')).toBe('noon');
    expect(slotPeriod('17:30')).toBe('evening');
    expect(slotPeriod('20:00')).toBe('night');
    expect(slotPeriod('23:00')).toBe('night');
  });
});

describe('weekdaySummary', () => {
  it('all seven on is «every day»', () => {
    expect(weekdaySummary([0, 1, 2, 3, 4, 5, 6])).toEqual({ kind: 'everyDay' });
  });

  it('a subset reports its count and sorted days', () => {
    expect(weekdaySummary([4, 0, 2])).toEqual({ kind: 'days', count: 3, days: [0, 2, 4] });
  });

  it('none on is flagged', () => {
    expect(weekdaySummary([])).toEqual({ kind: 'none' });
  });

  it('toggling one day off breaks «every day»', () => {
    const days = toggleWeekday([0, 1, 2, 3, 4, 5, 6], 6);
    expect(weekdaySummary(days)).toEqual({ kind: 'days', count: 6, days: [0, 1, 2, 3, 4, 5] });
    expect(weekdaySummary(toggleWeekday(days, 6))).toEqual({ kind: 'everyDay' });
  });

  it('orders the circles by calendar', () => {
    expect(weekdayOrder(true)).toEqual([0, 1, 2, 3, 4, 5, 6]);
    expect(weekdayOrder(false)).toEqual([1, 2, 3, 4, 5, 6, 0]);
  });
});

describe('validateForm', () => {
  const base = { ...emptyForm('2026-09-26'), title: 'Folic acid', dose: '400', unit: 'mcg' };

  it('accepts a valid form and emits MedicationInput', () => {
    const result = validateForm(base);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.input).toMatchObject({ title: 'Folic acid', endsOn: null, notes: null });
    }
  });

  it('requires a name', () => {
    const result = validateForm({ ...base, title: '  ' });
    expect(result).toEqual({ ok: false, errors: { title: 'titleRequired' } });
  });

  it('rejects duplicate times and empty weekdays', () => {
    const result = validateForm({ ...base, times: ['08:00', '08:00'], weekdays: [] });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.errors.times).toBe('timesDuplicate');
      expect(result.errors.weekdays).toBe('weekdaysRequired');
    }
  });

  it('until_date needs an end on or after the start', () => {
    const missing = validateForm({ ...base, duration: 'until_date', endsOn: null });
    expect(!missing.ok && missing.errors.endsOn).toBe('endsOnRequired');
    const early = validateForm({ ...base, duration: 'until_date', endsOn: '2026-09-01' });
    expect(!early.ok && early.errors.endsOn).toBe('endsOnBeforeStart');
  });

  it('keeps the amount within 1–10', () => {
    const result = validateForm({ ...base, amount: 11 });
    expect(!result.ok && result.errors.amount).toBe('amountRange');
  });
});

describe('mapServerErrors', () => {
  it('maps snake_case and indexed keys onto fields', () => {
    const { fields, unknown } = mapServerErrors({
      success: false,
      errors: { 'times.1': ['bad time'], ends_on: ['too early'], title: ['required'], foo: ['x'] },
    });
    expect(fields).toEqual({ times: 'bad time', endsOn: 'too early', title: 'required' });
    expect(unknown).toBe('x');
  });

  it('tolerates a body without errors', () => {
    expect(mapServerErrors({ message: 'nope' })).toEqual({ fields: {}, unknown: null });
  });
});

describe('navigation + durations', () => {
  it('returns to the origin', () => {
    expect(returnHref('home')).toBe('/home');
    expect(returnHref(undefined)).toBe('/reminders');
  });

  it('offers «until end of pregnancy» only in pregnancy mode', () => {
    expect(durationOptions(false, 'ongoing')).toEqual(['ongoing', 'until_date']);
    expect(durationOptions(true, 'ongoing')).toContain('pregnancy_end');
    expect(durationOptions(false, 'pregnancy_end')).toContain('pregnancy_end');
  });
});

describe('default duration', () => {
  it('starts a pregnant user on «تا پایان بارداری», everyone else on «بدون تاریخ پایان»', () => {
    expect(defaultDuration(true)).toBe('pregnancy_end');
    expect(defaultDuration(false)).toBe('ongoing');
    expect(emptyForm('2026-09-29', true).duration).toBe('pregnancy_end');
    expect(emptyForm('2026-09-29').duration).toBe('ongoing');
    expect(emptyForm('2026-09-29', true).endsOn).toBeNull();
  });

  it('a pregnancy_end medication validates without an end date', () => {
    const r = validateForm({ ...emptyForm('2026-09-29', true), title: 'Folic acid', dose: '400' });
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.input).toMatchObject({ duration: 'pregnancy_end', endsOn: null });
  });
});
