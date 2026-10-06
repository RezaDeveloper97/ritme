import { describe, expect, it } from 'vitest';

import { CHART_H, CHART_W, trendGeometry } from './chart';

describe('trendGeometry', () => {
  it('runs oldest → newest left to right and keeps lower values lower', () => {
    const g = trendGeometry([32, 18, 9], 15, 150);
    expect(g.points.map((p) => p.x)).toEqual([...g.points.map((p) => p.x)].sort((a, b) => a - b));
    expect(g.points[2]!.y).toBeGreaterThan(g.points[0]!.y);
    expect(g.band!.top).toBeLessThan(g.band!.bottom);
    expect(g.ticks.map((t) => t.value)).toEqual([15, 150]);
    for (const p of g.points) {
      expect(p.y).toBeGreaterThanOrEqual(0);
      expect(p.y).toBeLessThanOrEqual(CHART_H);
      expect(p.x).toBeLessThanOrEqual(CHART_W);
    }
  });

  it('centres a single point and survives a flat series without a range', () => {
    const one = trendGeometry([5], null, null);
    expect(one.points[0]!.x).toBe(CHART_W / 2);
    expect(one.band).toBeNull();
    const flat = trendGeometry([2, 2], null, null);
    expect(Number.isFinite(flat.points[0]!.y)).toBe(true);
  });
});
