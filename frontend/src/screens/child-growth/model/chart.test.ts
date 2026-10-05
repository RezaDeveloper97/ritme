import { describe, expect, it } from 'vitest';

import { bandPath, buildScale, medianPath, monthStep, niceStep, PLOT, pointCoords, visibleToMonth } from './chart';
import { checkForm, emptyForm, toMeasurementBody } from './form';
import type { GrowthPoint, GrowthReferenceRow } from './types';

const ref: GrowthReferenceRow[] = [
  { month: 0, p3: 2.4, p15: 2.8, p50: 3.2, p85: 3.7, p97: 4.2 },
  { month: 1, p3: 3.2, p15: 3.6, p50: 4.2, p85: 4.8, p97: 5.4 },
  { month: 2, p3: 3.9, p15: 4.5, p50: 5.1, p85: 5.9, p97: 6.6 },
];
const pts: GrowthPoint[] = [
  { id: 1, measuredOn: '2026-08-12', ageMonths: 2, value: 5, percentile: 42, inBand: true },
  { id: null, measuredOn: '2026-06-12', ageMonths: 0, value: 3.2, percentile: 47, inBand: true },
];

describe('growth chart geometry', () => {
  it('picks nice steps', () => {
    expect(niceStep(4)).toBe(1);
    expect(niceStep(8)).toBe(2);
    expect(niceStep(40)).toBe(10);
    expect(niceStep(0)).toBe(1);
    expect(monthStep(12)).toBe(1);
    expect(monthStep(18)).toBe(3);
    expect(monthStep(60)).toBe(12);
  });

  it('fits the band and the points, months left → right', () => {
    const s = buildScale(0, 2, ref, pts);
    expect(s.minValue).toBeLessThanOrEqual(2.4);
    expect(s.maxValue).toBeGreaterThanOrEqual(6.6);
    expect(s.x(0)).toBe(PLOT.left);
    expect(s.x(2)).toBe(PLOT.right);
    expect(s.y(s.minValue)).toBe(PLOT.bottom);
    expect(s.y(s.maxValue)).toBe(PLOT.top);
    expect(s.xTicks).toEqual([0, 1, 2]);
    expect(s.yTicks[0]).toBe(s.minValue);
    expect(s.yTicks.at(-1)).toBe(s.maxValue);
  });

  it('draws a closed band, an open median and sorted points', () => {
    const s = buildScale(0, 2, ref, pts);
    const band = bandPath(ref, s);
    expect(band.startsWith('M')).toBe(true);
    expect(band.endsWith('Z')).toBe(true);
    expect(band.split('L')).toHaveLength(6);
    expect(medianPath(ref, s).split('L')).toHaveLength(3);
    expect(pointCoords(pts, s).map((p) => p.point.ageMonths)).toEqual([0, 2]);
    expect(bandPath([], s)).toBe('');
  });

  it('works with points only (no reference)', () => {
    const s = buildScale(0, 12, [], [pts[0]!]);
    expect(s.maxValue).toBeGreaterThan(s.minValue);
    expect(s.xTicks).toHaveLength(13);
  });
});

describe('visible range', () => {
  it('shows the age plus 3 months, at least 6, within the server range', () => {
    expect(visibleToMonth({ fromMonth: 0, toMonth: 12, points: [] })).toBe(6);
    expect(visibleToMonth({ fromMonth: 0, toMonth: 12, points: [{ ...pts[0]!, ageMonths: 3.4 }] })).toBe(7);
    expect(visibleToMonth({ fromMonth: 0, toMonth: 12, points: [{ ...pts[0]!, ageMonths: 11 }] })).toBe(12);
  });
});

describe('measurement form', () => {
  it('needs one value in range', () => {
    const f = emptyForm('2026-09-23');
    expect(checkForm(f)).toEqual({ field: 'all', kind: 'empty' });
    expect(checkForm({ ...f, weightKg: '80' })).toEqual({ field: 'weightKg', kind: 'range' });
    expect(checkForm({ ...f, headCm: '40.5' })).toBeNull();
  });

  it('sends every key, null when empty', () => {
    expect(toMeasurementBody({ measuredOn: '2026-09-23', weightKg: '6.1', lengthCm: '', headCm: '40' })).toEqual({
      measured_on: '2026-09-23',
      weight_kg: 6.1,
      length_cm: null,
      head_cm: 40,
    });
  });
});
