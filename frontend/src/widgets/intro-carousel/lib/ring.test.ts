import { describe, expect, it } from 'vitest';

import {
  CYCLE_DAYS,
  cycleDotKind,
  isCycleDotFilled,
  ringPoint,
  trimesterOf,
} from './ring';

describe('ringPoint', () => {
  it('starts at 12 o’clock and runs clockwise (artboard coordinates)', () => {
    expect(ringPoint(0, CYCLE_DAYS)).toEqual({ x: 150, y: 22 });
    // Values copied from nbl_Splash.dc.html.
    expect(ringPoint(1, CYCLE_DAYS)).toEqual({ x: 177.5, y: 25 });
    expect(ringPoint(22, CYCLE_DAYS)).toEqual({ x: 22.2, y: 143.1 });
    // nbl_Intro_2: week 32 marker.
    expect(ringPoint(31, 40)).toEqual({ x: 23.6, y: 130 });
  });
});

describe('cycleDotKind', () => {
  it('lays out the illustrative 29-day cycle', () => {
    const kinds = Array.from({ length: CYCLE_DAYS }, (_, i) => cycleDotKind(i));
    expect(kinds.filter((k) => k === 'period')).toHaveLength(5);
    expect(kinds.filter((k) => k === 'fertile')).toHaveLength(5);
    expect(kinds.filter((k) => k === 'ovulation')).toHaveLength(1);
    expect(kinds.filter((k) => k === 'pms')).toHaveLength(6);
    expect(kinds[14]).toBe('ovulation');
  });

  it('fills past days and ovulation only', () => {
    expect(isCycleDotFilled(0, 0)).toBe(true);
    expect(isCycleDotFilled(1, 0)).toBe(false);
    expect(isCycleDotFilled(14, 0)).toBe(true);
    expect(isCycleDotFilled(22, 22)).toBe(true);
    expect(isCycleDotFilled(23, 22)).toBe(false);
  });
});

describe('trimesterOf', () => {
  it('splits 40 weeks into 13 / 14 / 13', () => {
    const t = Array.from({ length: 40 }, (_, i) => trimesterOf(i));
    expect(t.filter((x) => x === 1)).toHaveLength(13);
    expect(t.filter((x) => x === 2)).toHaveLength(14);
    expect(t.filter((x) => x === 3)).toHaveLength(13);
  });
});
