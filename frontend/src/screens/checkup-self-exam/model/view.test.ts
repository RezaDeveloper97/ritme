import { describe, expect, it } from 'vitest';

import { adherence, daysUntilWindow, doneThisMonth, findSelfExam } from './view';

describe('checkup-self-exam view', () => {
  it('counts distinct months in the last 12', () => {
    const recs = ['2026-09-02', '2026-09-20', '2026-08-01', '2025-10-05', '2025-09-30'].map((doneOn) => ({ doneOn }));
    expect(adherence(recs, '2026-09-26')).toBe(3);
    expect(doneThisMonth(recs, '2026-09-26')).toBe(true);
    expect(doneThisMonth(recs, '2026-10-01')).toBe(false);
  });

  it('computes days until the best window', () => {
    expect(daysUntilWindow(3, 28, 7, 10)).toBe(4);
    expect(daysUntilWindow(8, 28, 7, 10)).toBe(0);
    expect(daysUntilWindow(20, 28, 7, 10)).toBe(15);
  });

  it('finds the self-exam type by key', () => {
    expect(findSelfExam(undefined)).toBeNull();
  });
});
