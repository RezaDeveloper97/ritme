import { describe, expect, it } from 'vitest';

import { homeMonthSet, shiftMonth } from './homeMonths';

describe('shiftMonth', () => {
  it('rolls the year over both ways', () => {
    expect(shiftMonth({ year: 2026, month: 12 }, 1)).toEqual({ year: 2027, month: 1 });
    expect(shiftMonth({ year: 2026, month: 1 }, -1)).toEqual({ year: 2025, month: 12 });
    expect(shiftMonth({ year: 2026, month: 6 }, 0)).toEqual({ year: 2026, month: 6 });
  });
});

describe('homeMonthSet', () => {
  const cur = { year: 2026, month: 11 };
  it('is exactly prev/current/next without dates', () => {
    expect(homeMonthSet(cur)).toEqual([
      { year: 2026, month: 10 }, { year: 2026, month: 11 }, { year: 2026, month: 12 },
    ]);
  });
  it('adds no request for days inside prev/current/next (month boundary days)', () => {
    expect(homeMonthSet(cur, ['2026-10-01', '2026-10-31', '2026-12-01', '2026-12-31'])).toHaveLength(3);
  });
  it('rolls over the year at the edges', () => {
    expect(homeMonthSet({ year: 2026, month: 12 })).toEqual([
      { year: 2026, month: 11 }, { year: 2026, month: 12 }, { year: 2027, month: 1 },
    ]);
    expect(homeMonthSet({ year: 2026, month: 1 })[0]).toEqual({ year: 2025, month: 12 });
  });
  it('Esfand / leap day (Gregorian late Feb–Mar) stays in the set', () => {
    const feb = { year: 2028, month: 2 };
    expect(homeMonthSet(feb, ['2028-02-29', '2028-03-01'])).toHaveLength(3);
  });
  it('falls back: a day outside the window adds its month (long cycle / late period / pregnancy)', () => {
    const set = homeMonthSet(cur, ['2026-08-20', '2027-02-03']);
    expect(set).toContainEqual({ year: 2026, month: 8 });
    expect(set).toContainEqual({ year: 2027, month: 2 });
    expect(set).toHaveLength(5);
  });
  it('ignores malformed dates and duplicates', () => {
    expect(homeMonthSet(cur, ['garbage', '2026-13-01', '2026-11-05', '2026-11-06'])).toHaveLength(3);
  });
});
