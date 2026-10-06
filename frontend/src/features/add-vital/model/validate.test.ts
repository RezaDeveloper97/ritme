import { describe, expect, it } from 'vitest';

import { measuredAtBody, parseLocaleNumber, tehranMs, tehranNow, validateBp, validateGlucose, validateHr, validateMeasuredAt, validateNote } from './validate';

describe('add-vital validation mirrors the server', () => {
  it('blood pressure ranges and systolic > diastolic', () => {
    expect(validateBp(118, 76, 70)).toEqual({});
    expect(validateBp(40, 76, null).systolic).toEqual({ key: 'range', min: 50, max: 300 });
    expect(validateBp(118, 201, null).diastolic).toEqual({ key: 'range', min: 30, max: 200 });
    expect(validateBp(118, 76, 20).pulse).toEqual({ key: 'range', min: 30, max: 250 });
    expect(validateBp(80, 80, null).systolic).toEqual({ key: 'systolicGtDiastolic' });
    expect(validateBp(null, 80, null).systolic?.key).toBe('range');
    expect(validateBp(118.5, 76, null).systolic?.key).toBe('range');
  });

  it('glucose range depends on the unit typed', () => {
    expect(validateGlucose(94, 'mg_dl')).toEqual({});
    expect(validateGlucose(19, 'mg_dl').value).toEqual({ key: 'range', min: 20, max: 600 });
    expect(validateGlucose(5.2, 'mmol_l')).toEqual({});
    expect(validateGlucose(94, 'mmol_l').value).toEqual({ key: 'range', min: 1.1, max: 33.3 });
  });

  it('heart rate 30–250, whole numbers', () => {
    expect(validateHr(72)).toEqual({});
    expect(validateHr(251).bpm?.key).toBe('range');
  });

  it('measuring time: not in the future, not older than two years', () => {
    const now = tehranMs('2026-10-06', '10:00');
    expect(validateMeasuredAt(null, now)).toEqual({});
    expect(validateMeasuredAt({ date: '2026-10-06', time: '10:00' }, now)).toEqual({});
    expect(validateMeasuredAt({ date: '2026-10-06', time: '10:05' }, now).measured_at).toEqual({ key: 'future' });
    expect(validateMeasuredAt({ date: '2024-10-01', time: '10:00' }, now).measured_at).toEqual({ key: 'tooOld' });
    expect(measuredAtBody({ date: '2026-10-06', time: '08:10' })).toBe('2026-10-06 08:10');
    expect(measuredAtBody(null)).toBeNull();
  });

  it('note length', () => {
    expect(validateNote('a'.repeat(500))).toEqual({});
    expect(validateNote('a'.repeat(501)).note?.key).toBe('noteTooLong');
  });

  it('parses any digit script and decimal separator', () => {
    expect(parseLocaleNumber('۱۱۸', 0)).toBe(118);
    expect(parseLocaleNumber('۵٫۲', 1)).toBe(5.2);
    expect(parseLocaleNumber('5,2', 1)).toBe(5.2);
    expect(parseLocaleNumber('٩٤', 0)).toBe(94);
    expect(parseLocaleNumber('5.27', 1)).toBe(5.2);
    expect(parseLocaleNumber('', 0)).toBeNull();
    expect(parseLocaleNumber('abc', 0)).toBeNull();
  });

  it('reads the Tehran wall clock', () => {
    expect(tehranNow(new Date('2026-10-06T06:55:00Z'))).toEqual({ date: '2026-10-06', time: '10:25' });
  });
});
