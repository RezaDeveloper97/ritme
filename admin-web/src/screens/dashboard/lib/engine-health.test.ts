import { describe, expect, it } from 'vitest';

import { bucketRows, engineReadable } from './engine-health';

describe('engine health', () => {
  it('orders buckets and hides suppressed ones instead of showing 0', () => {
    const rows = bucketRows(
      { of: 120, suppressed: false, buckets: [
        { key: 'high', pct: 60, count: 72, suppressed: false },
        { key: 'low', pct: null, count: null, suppressed: true },
        { key: 'medium', pct: 40, count: 48, suppressed: false },
      ] },
      ['low', 'medium', 'high'],
    );
    expect(rows.map((r) => r.key)).toEqual(['low', 'medium', 'high']);
    expect(rows[0]).toMatchObject({ hidden: true, pct: null });
    expect(rows[2]).toMatchObject({ hidden: false, pct: 60 });
  });

  it('hides every bucket of a suppressed distribution', () => {
    const rows = bucketRows({ of: 4, suppressed: true, buckets: [{ key: 'low', pct: 50, count: 2, suppressed: false }] }, ['low']);
    expect(rows[0]?.hidden).toBe(true);
    expect(bucketRows(null, [])).toEqual([]);
  });

  it('reads engine accuracy only when not suppressed', () => {
    expect(engineReadable({ suppressed: false, mean_error_days: 1.2 })).toBe(true);
    expect(engineReadable({ suppressed: true, mean_error_days: 1.2 })).toBe(false);
    expect(engineReadable({ suppressed: false, mean_error_days: null })).toBe(false);
  });
});
