import { describe, expect, it } from 'vitest';

import { CHART_W, PAD_X, labelIndexes, linePath, makeScale, slotX, tickStep } from './chart';

describe('report chart geometry', () => {
  it('nice tick steps', () => {
    expect(tickStep(80)).toBe(20);
    expect(tickStep(4)).toBe(1);
    expect(tickStep(1.2)).toBe(0.5);
  });

  it('the scale always shows the band and orders values top-down', () => {
    const s = makeScale([121, 78, 132, 86], [80, 120], 170);
    expect(s.min).toBeLessThanOrEqual(78);
    expect(s.max).toBeGreaterThanOrEqual(132);
    expect(s.y(s.max)).toBeLessThan(s.y(s.min));
    expect(s.ticks[0]).toBe(s.min);
  });

  it('slots run left → right inside the plot', () => {
    expect(slotX(0, 7)).toBeGreaterThan(PAD_X);
    expect(slotX(6, 7)).toBeLessThan(CHART_W);
    expect(slotX(0, 7)).toBeLessThan(slotX(1, 7));
  });

  it('a null breaks the line', () => {
    expect(linePath([{ x: 1, y: 2 }, null, { x: 3, y: 4 }, { x: 5, y: 6 }])).toBe('M1 2M3 4L5 6');
  });

  it('x labels keep the ends', () => {
    expect(labelIndexes(5, 7)).toEqual([0, 1, 2, 3, 4]);
    const idx = labelIndexes(30, 7);
    expect(idx[0]).toBe(0);
    expect(idx[idx.length - 1]).toBe(29);
    expect(idx.length).toBeLessThanOrEqual(7);
  });
});
