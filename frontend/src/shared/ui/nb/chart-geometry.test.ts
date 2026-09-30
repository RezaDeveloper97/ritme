import { describe, expect, it } from 'vitest';

import { bandPath, barRects, domainOf, linePath, ringDash, xAt, yAt, type PlotBox } from './chart-geometry';

const box: PlotBox = { width: 100, height: 50, pad: [0, 0, 0, 0] };

describe('chart geometry', () => {
  it('derives a domain over finite values and widens a flat one', () => {
    expect(domainOf([[1, null, 5], [3]])).toEqual({ min: 1, max: 5 });
    expect(domainOf([[2, 2]])).toEqual({ min: 1, max: 3 });
    expect(domainOf([[2, 9]], { min: 0 })).toEqual({ min: 0, max: 9 });
  });

  it('lays points left → right and values bottom → top', () => {
    expect(xAt(0, 3, box)).toBe(0);
    expect(xAt(2, 3, box)).toBe(100);
    expect(xAt(0, 1, box)).toBe(50);
    expect(yAt(0, { min: 0, max: 10 }, box)).toBe(50);
    expect(yAt(10, { min: 0, max: 10 }, box)).toBe(0);
  });

  it('breaks the line at null (a missed day is a gap)', () => {
    const d = linePath([0, 10, null, 10], { min: 0, max: 10 }, box);
    expect(d.match(/M/g)).toHaveLength(2);
    expect(d.startsWith('M0 50L33.33 0')).toBe(true);
    expect(linePath([null], { min: 0, max: 1 }, box)).toBe('');
  });

  it('closes a band from upper forwards and lower backwards', () => {
    const d = bandPath([0, 0], [10, 10], { min: 0, max: 10 }, box);
    expect(d).toBe('M0 0L100 0L100 50L0 50Z');
  });

  it('draws bars from the baseline, clamped to the max', () => {
    const [a, b] = barRects([5, 20], 10, box, 0);
    expect(a).toEqual({ x: 0, y: 25, width: 50, height: 25 });
    expect(b.height).toBe(50);
  });

  it('dashes a ring arc for a fraction of the circumference', () => {
    const { dash, circumference } = ringDash(0.5, 10);
    expect(circumference).toBeCloseTo(62.83, 1);
    expect(dash.split(' ')[0]).toBe('31.42');
    expect(ringDash(2, 10).dash.split(' ')[0]).toBe(String(circumference));
  });
});
