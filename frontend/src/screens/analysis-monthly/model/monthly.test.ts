import { describe, expect, it } from 'vitest';

import { deltaTone, formatYm, neighbour, parseYm, placeMonth, resolveMonth, signed, type MonthBounds } from './monthly';

const bounds: MonthBounds = { current: { year: 1405, month: 7 }, earliest: { year: 1403, month: 8 } };

describe('monthly model', () => {
  it('parses and formats ym', () => {
    expect(parseYm('1405-07')).toEqual({ year: 1405, month: 7 });
    expect(parseYm('1405-13')).toBeNull();
    expect(parseYm('1405-7')).toBeNull();
    expect(formatYm({ year: 2026, month: 9 })).toBe('2026-09');
  });

  it('keeps the stepper inside the bounds (no future months)', () => {
    expect(neighbour({ year: 1405, month: 7 }, 1, bounds)).toBeNull();
    expect(neighbour({ year: 1405, month: 7 }, -1, bounds)).toEqual({ year: 1405, month: 6 });
    expect(neighbour({ year: 1405, month: 1 }, -1, bounds)).toEqual({ year: 1404, month: 12 });
    expect(neighbour({ year: 1403, month: 8 }, -1, bounds)).toBeNull();
    expect(placeMonth({ year: 1405, month: 8 }, bounds)).toBe('future');
    expect(placeMonth({ year: 1403, month: 7 }, bounds)).toBe('too_old');
    expect(placeMonth({ year: 1404, month: 1 }, bounds)).toBe('ok');
  });

  it('re-expresses a month from the other calendar in the reader’s calendar', () => {
    // 1 Mehr 1405 = 23 September 2026.
    expect(resolveMonth({ year: 1405, month: 7 }, 'jalali', 'en')).toEqual({ year: 2026, month: 9 });
    // 1 September 2026 = 10 Shahrivar 1405.
    expect(resolveMonth({ year: 2026, month: 9 }, 'gregorian', 'fa')).toEqual({ year: 1405, month: 6 });
    expect(resolveMonth({ year: 1405, month: 7 }, 'jalali', 'fa')).toEqual({ year: 1405, month: 7 });
    // Esfand → Farvardin across the year (the jalaliday setter pitfall).
    expect(resolveMonth({ year: 2026, month: 3 }, 'gregorian', 'fa')).toEqual({ year: 1404, month: 12 });
  });

  it('colours deltas', () => {
    expect(deltaTone('days_logged', 6)).toBe('data');
    expect(deltaTone('good_mood', -5)).toBe('warm');
    expect(deltaTone('cycle_length', -1)).toBe('data');
    expect(deltaTone('sleep', 0)).toBe('flat');
    expect(deltaTone('blood_pressure', { systolic: 2, diastolic: 1 })).toBe('warm');
    expect(deltaTone('blood_pressure', { systolic: 0, diastolic: 0 })).toBe('flat');
  });

  it('signs numbers', () => {
    expect(signed(-1, 0)).toEqual({ sign: '−', abs: '1' });
    expect(signed(0.3, 1)).toEqual({ sign: '+', abs: '0.3' });
    expect(signed(2, 1)).toEqual({ sign: '+', abs: '2' });
    expect(signed(0.01, 1)).toEqual({ sign: '', abs: '0' });
  });
});
