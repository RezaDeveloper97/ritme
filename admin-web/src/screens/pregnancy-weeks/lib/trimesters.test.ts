import { describe, expect, it } from 'vitest';

import { byTrimester } from './trimesters';

describe('byTrimester', () => {
  it('splits 1…40 into 13 / 14 / 13 weeks and keeps weeks above 40 apart', () => {
    const weeks = Array.from({ length: 40 }, (_, i) => ({ week: i + 1 }));
    const bands = byTrimester([...weeks, { week: 41 }]);
    expect(bands.map((b) => [b.key, b.weeks.length])).toEqual([
      ['first', 13],
      ['second', 14],
      ['third', 13],
      ['beyond', 1],
    ]);
  });
});
