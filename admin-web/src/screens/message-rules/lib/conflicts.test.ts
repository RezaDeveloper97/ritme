import { describe, expect, it } from 'vitest';

import { ruleSchema, type Conflict } from '../api/rules';
import { describeRange, toConflictView } from './conflicts';

const range = (o: object) => ({ absent: false, present: true, min: null, max: null, min_open: false, max_open: false, values: null, contains: null, any_of: null, not: [], ...o }) as never;

describe('conflict panel mapping', () => {
  it('describes number, value and set overlaps', () => {
    expect(describeRange(range({ fact: 'cycle_day', kind: 'number', min: 5, max: 9 }))).toBe('cycle_day ≥ 5 ≤ 9');
    expect(describeRange(range({ fact: 'pregnancy_week', kind: 'number', min: 12, max: 12 }))).toBe('pregnancy_week = 12');
    expect(describeRange(range({ fact: 'mode', kind: 'value', values: ['cycle', 'ttc'], not: ['teen'] }))).toBe('mode ∈ cycle, ttc ∖ teen');
    expect(describeRange(range({ fact: 'tags', kind: 'set', contains: ['x'] }))).toBe('tags ∋ x');
  });
  it('maps winner, slot, tie-break and the overlap of a conflict', () => {
    const c = {
      rule_id: 7, title: 'Other', priority: 40, kind: 'engine', winner: 'other', safety: false, slot: 'smart_tip', tie_break: true, co_shown: false,
      overlap: { starts_at: '2026-11-01 00:00', ends_at: null, hours: [[8, 12]], audience_percent: 25, facts: [range({ fact: 'cycle_day', kind: 'number', min: 1, max: 5 })] },
    } as unknown as Conflict;
    const v = toConflictView(c);
    expect(v).toMatchObject({ ruleId: 7, winner: 'other', slot: 'smart_tip', tieBreak: true, schedule: '2026-11-01 00:00 → …', hours: '08:00–12:00', audience: 25 });
    expect(v.facts).toEqual(['cycle_day ≥ 1 ≤ 5']);
  });
  it('accepts a v1 conflict without the v2 fields', () => {
    const v = toConflictView({ rule_id: 1, title: 't', priority: 1, kind: 'engine', winner: 'this', safety: false } as unknown as Conflict);
    expect(v.facts).toEqual([]);
    expect(v.schedule).toBeNull();
    expect(typeof ruleSchema.parse).toBe('function');
  });
});
