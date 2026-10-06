import { describe, expect, it } from 'vitest';

import { formatLabValue, formatReference } from './format';
import { parseLabNumber, rangeGeometry, splitMarkers, stateTone } from './range';
import type { LabMarker } from './types';

describe('rangeGeometry', () => {
  it('puts an in-range value inside the band and a low one before it', () => {
    const inRange = rangeGeometry(2.1, 0.4, 4)!;
    expect(inRange.marker).toBeGreaterThan(inRange.bandStart);
    expect(inRange.marker).toBeLessThan(inRange.bandEnd);
    const low = rangeGeometry(9, 15, 150)!;
    expect(low.marker).toBeLessThan(low.bandStart);
    const high = rangeGeometry(300, 15, 150)!;
    expect(high.marker).toBeGreaterThan(high.bandEnd);
    expect(high.marker).toBeLessThanOrEqual(97);
  });

  it('handles one-sided ranges and missing data', () => {
    const upper = rangeGeometry(3, null, 5)!;
    expect(upper.bandStart).toBe(0);
    expect(upper.marker).toBeLessThan(upper.bandEnd);
    const lower = rangeGeometry(60, 40, null)!;
    expect(lower.bandEnd).toBe(100);
    expect(rangeGeometry(null, 1, 2)).toBeNull();
    expect(rangeGeometry(5, null, null)).toBeNull();
  });
});

describe('lab marker helpers', () => {
  const m = (over: Partial<LabMarker>): LabMarker => ({
    id: 1, code: null, name: 'x', printedName: 'x', subtitle: null, value: 1, valueText: null, unit: null,
    reference: { low: null, high: null, text: null, source: null }, state: 'normal', stateLabel: '',
    attention: false, confidence: null, lowConfidence: false, source: 'extracted', ...over,
  });

  it('orders attention first (out of range before borderline)', () => {
    const { attention, normal, unknown } = splitMarkers([
      m({ id: 1, state: 'borderline_low', attention: true }),
      m({ id: 2 }),
      m({ id: 3, state: 'low', attention: true }),
      m({ id: 4, state: 'unknown' }),
    ]);
    expect(attention.map((x) => x.id)).toEqual([3, 1]);
    expect(normal.map((x) => x.id)).toEqual([2]);
    expect(unknown.map((x) => x.id)).toEqual([4]);
    expect(stateTone('low')).toBe('warm');
    expect(stateTone('normal')).toBe('data');
  });

  it('parses Persian and Latin numbers like the server', () => {
    expect(parseLabNumber('۱۱٫۲')).toBe(11.2);
    expect(parseLabNumber(' 92 ')).toBe(92);
    expect(parseLabNumber('-0.5')).toBe(-0.5);
    expect(parseLabNumber('')).toBeNull();
    expect(parseLabNumber('12abc')).toBeNull();
    expect(parseLabNumber('1,200')).toBeNull();
  });

  it('formats values and ranges in the locale digits', () => {
    expect(formatLabValue({ value: 11.2, valueText: null }, 'fa')).toBe('۱۱٫۲');
    expect(formatLabValue({ value: null, valueText: 'Negative' }, 'en')).toBe('Negative');
    expect(formatReference({ low: 12, high: 16, text: null, source: 'sheet' }, 'en')).toBe('12–16');
    expect(formatReference({ low: null, high: null, text: '0.4–4', source: 'sheet' }, 'fa')).toBe('۰٫۴–۴');
    expect(formatReference({ low: null, high: null, text: null, source: null }, 'fa')).toBeNull();
  });
});
