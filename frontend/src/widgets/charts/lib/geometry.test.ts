import { describe, expect, it } from 'vitest';

import { axisTicks, cellOpacity, heightShare, maxOf, slot, stackSegments } from './geometry';

describe('chart geometry', () => {
  it('scales bar heights with a floor', () => {
    expect(heightShare(0, 10)).toBe(0);
    expect(heightShare(5, 10)).toBe(0.5);
    expect(heightShare(0.1, 10)).toBe(0.08);
    expect(heightShare(20, 10)).toBe(1);
    expect(heightShare(3, 0)).toBe(0);
  });

  it('lays out slots and stacks', () => {
    expect(slot(0, 4, 100, 4)).toEqual({ x: 0, w: 22 });
    expect(slot(3, 4, 100, 4)).toEqual({ x: 78, w: 22 });
    expect(stackSegments([5, 5, 10], 100)).toEqual([
      { x: 0, w: 25 },
      { x: 25, w: 25 },
      { x: 50, w: 50 },
    ]);
    expect(stackSegments([0, 0], 100)).toEqual([
      { x: 0, w: 0 },
      { x: 0, w: 0 },
    ]);
  });

  it('maps shares to cell opacity and picks axis ticks', () => {
    expect(cellOpacity(0)).toBe(0.12);
    expect(cellOpacity(1)).toBe(1);
    expect(cellOpacity(2)).toBe(1);
    expect(axisTicks(29)).toEqual([0, 14, 28]);
    expect(axisTicks(2)).toEqual([0, 1]);
    expect(axisTicks(0)).toEqual([]);
    expect(maxOf([null, 3, 7, Number.NaN])).toBe(7);
  });
});
