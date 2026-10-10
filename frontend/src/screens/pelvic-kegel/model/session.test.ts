import { describe, expect, it } from 'vitest';

import { durationSeconds, phaseAt, phaseCount, repBarFilled, setsDone } from './session';

const PLAN = { holdSec: 5, restSec: 5, reps: 10, sets: 3 };

describe('kegel session', () => {
  it('alternates hold and relax through reps and sets', () => {
    expect(phaseCount(PLAN)).toBe(60);
    expect(phaseAt(0, PLAN)).toEqual({ kind: 'hold', set: 1, rep: 1, ms: 5000 });
    expect(phaseAt(1, PLAN)).toEqual({ kind: 'relax', set: 1, rep: 1, ms: 5000 });
    expect(phaseAt(20, PLAN)).toEqual({ kind: 'hold', set: 2, rep: 1, ms: 5000 });
    // The board: set 3 of 3, rep 7 of 10, squeezing.
    expect(phaseAt(52, PLAN)).toMatchObject({ kind: 'hold', set: 3, rep: 7 });
  });

  it('fills the rep bar when a squeeze ends', () => {
    expect(repBarFilled(52, PLAN)).toBe(6);
    expect(repBarFilled(53, PLAN)).toBe(7);
    expect(repBarFilled(0, PLAN)).toBe(0);
    expect(repBarFilled(60, PLAN)).toBe(10);
  });

  it('counts whole sets only', () => {
    expect(setsDone(19, PLAN)).toBe(1);
    expect(setsDone(18, PLAN)).toBe(0);
    expect(setsDone(60, PLAN)).toBe(3);
  });

  it('clamps the saved duration', () => {
    expect(durationSeconds(0, 200)).toBe(1);
    expect(durationSeconds(0, 300_400)).toBe(300);
    expect(durationSeconds(0, 9_000_000)).toBe(3600);
  });
});
