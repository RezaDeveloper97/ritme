import { describe, expect, it } from 'vitest';

import type { IvfScan } from '@/entities/ivf';

import { checkForm, emptyCounts, formFrom, growthWindow, isDirty, parseDecimal, scanDate, stimDayOn } from './form';

const scan: IvfScan = {
  date: '2026-10-03',
  stimDay: 7,
  right: { lt_10: 3, '10_14': 4, '15_17': 2, '18_plus': 0 },
  left: { lt_10: 2, '10_14': 5, '15_17': 3, '18_plus': 1 },
  endometriumMm: '8.5',
  e2: '1250.00',
  e2Unit: 'pg_ml',
  notes: 'kept',
};

describe('ivf scan form', () => {
  it('starts from the saved scan, trimming padded decimals', () => {
    const form = formFrom(scan);
    expect(form.right['10_14']).toBe(4);
    expect(form.endometrium).toBe('8.5');
    expect(form.e2).toBe('1250');
    expect(formFrom(scan, (v) => v.replace('.', '٫')).endometrium).toBe('8٫5');
    expect(formFrom(undefined)).toEqual({ right: emptyCounts(), left: emptyCounts(), endometrium: '', e2: '', e2Unit: 'pg_ml' });
  });

  it('parses decimals in any digit script', () => {
    expect(parseDecimal('', 40)).toEqual({ ok: true, value: null });
    expect(parseDecimal('۸٫۵', 40)).toEqual({ ok: true, value: 8.5 });
    expect(parseDecimal('7,2', 40)).toEqual({ ok: true, value: 7.2 });
    expect(parseDecimal('41', 40)).toEqual({ ok: false });
    expect(parseDecimal('abc', 40)).toEqual({ ok: false });
  });

  it('builds the PUT body and keeps the notes', () => {
    const form = { ...formFrom(scan), endometrium: '۹', e2: '' };
    const { input, errors } = checkForm(form, '2026-10-03', 'kept');
    expect(errors).toEqual({ endometrium: false, e2: false });
    expect(input).toMatchObject({ date: '2026-10-03', endometriumMm: 9, e2: null, e2Unit: null, notes: 'kept' });
    expect(checkForm({ ...form, endometrium: '99' }, '2026-10-03', null).input).toBeNull();
  });

  it('compares numbers, not text, for dirtiness', () => {
    const base = formFrom(scan);
    expect(isDirty(base, { ...base, endometrium: '8.50' })).toBe(false);
    expect(isDirty(base, { ...base, endometrium: '۸٫۵' })).toBe(false);
    expect(isDirty(base, { ...base, right: { ...base.right, lt_10: 4 } })).toBe(true);
    expect(isDirty(base, { ...base, e2Unit: 'pmol_l' })).toBe(true);
  });

  it('derives the stimulation day and the accepted date', () => {
    expect(stimDayOn('2026-10-03', '2026-09-27')).toBe(7);
    expect(stimDayOn('2026-09-26', '2026-09-27')).toBeNull();
    expect(stimDayOn('2026-10-03', null)).toBeNull();
    expect(scanDate(undefined, '2026-10-04')).toBe('2026-10-04');
    expect(scanDate('2026-10-09', '2026-10-04')).toBe('2026-10-04');
    expect(scanDate('2026-10-01', '2026-10-04')).toBe('2026-10-01');
    expect(scanDate('nope', '2026-10-04')).toBe('2026-10-04');
  });

  it('keeps the last six growth points, oldest first', () => {
    const points = Array.from({ length: 8 }, (_, i) => ({
      date: `2026-10-0${8 - i}`,
      stimDay: 8 - i,
      mid: i,
      lead: 0,
    }));
    const window = growthWindow(points);
    expect(window).toHaveLength(6);
    expect(window[0]?.date).toBe('2026-10-03');
    expect(window[5]?.date).toBe('2026-10-08');
  });
});
