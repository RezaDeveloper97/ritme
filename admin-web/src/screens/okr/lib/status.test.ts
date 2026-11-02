import { describe, expect, it } from 'vitest';

import { elapsedPct, paceFloor, statusHint } from './status';

describe('OKR status copy', () => {
  it('computes the elapsed share of a period', () => {
    expect(elapsedPct('2026-11-01', '2026-11-10', new Date('2026-10-20T12:00:00Z'))).toBe(0);
    expect(elapsedPct('2026-11-01', '2026-11-10', new Date('2026-11-05T12:00:00Z'))).toBe(50);
    expect(elapsedPct('2026-11-01', '2026-11-10', new Date('2026-12-01T00:00:00Z'))).toBe(100);
    expect(elapsedPct('bad', '2026-11-10', new Date())).toBeNull();
  });

  it('keeps the 10-point pace slack and never goes below zero', () => {
    expect(paceFloor(50)).toBe(40);
    expect(paceFloor(4)).toBe(0);
  });

  it('explains an at-risk key result by its pace and falls back to the status', () => {
    expect(statusHint('at_risk', 20, 60)).toEqual({ key: 'behind', elapsed: 60, floor: 50 });
    expect(statusHint('on_track', 55, 60).key).toBe('on_track');
    expect(statusHint('no_data', null, 60).key).toBe('no_data');
    expect(statusHint('at_risk', null, null).key).toBe('at_risk');
  });
});
