import { describe, expect, it } from 'vitest';

import { countParts, countsLine, ringFraction } from './counts';

const label = (key: string, n: number) => `${n} ${key}`;

describe('countsLine', () => {
  it('joins all three non-zero parts in order', () => {
    expect(countsLine({ total: 6, upToDate: 4, due: 1, overdue: 1 }, label, ' · ')).toBe(
      '4 countUpToDate · 1 countDue · 1 countOverdue',
    );
  });

  it('omits zero parts', () => {
    expect(countsLine({ total: 6, upToDate: 4, due: 0, overdue: 2 }, label, ' · ')).toBe(
      '4 countUpToDate · 2 countOverdue',
    );
    expect(countParts({ total: 2, upToDate: 0, due: 2, overdue: 0 })).toEqual([
      { key: 'countDue', count: 2 },
    ]);
  });

  it('is empty when every count is zero', () => {
    expect(countsLine({ total: 0, upToDate: 0, due: 0, overdue: 0 }, label, ' · ')).toBe('');
  });
});

describe('ringFraction', () => {
  it('clamps and handles zero total', () => {
    expect(ringFraction({ total: 0, upToDate: 0, due: 0, overdue: 0 })).toBe(0);
    expect(ringFraction({ total: 4, upToDate: 2, due: 2, overdue: 0 })).toBe(0.5);
  });
});
