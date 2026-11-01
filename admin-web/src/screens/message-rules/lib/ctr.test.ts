import { describe, expect, it } from 'vitest';

import { groupByVersion } from './ctr';

const s = (version: number, variant: string, impressions: number, clicks: number) => ({ version, variant, impressions, clicks, dismissals: 0, goals: 0, ctr: 0 });

describe('CTR per version and variant', () => {
  it('groups by version newest first with totals and a null CTR without impressions', () => {
    const rows = groupByVersion([s(2, 'b', 50, 10), s(3, 'a', 0, 0), s(2, 'a', 50, 5)]);
    expect(rows.map((r) => r.version)).toEqual([3, 2]);
    expect(rows[0]?.ctr).toBeNull();
    expect(rows[1]).toMatchObject({ impressions: 100, clicks: 15, ctr: 0.15 });
    expect(rows[1]?.variants.map((v) => [v.variant, v.ctr])).toEqual([['a', 0.1], ['b', 0.2]]);
  });
});
