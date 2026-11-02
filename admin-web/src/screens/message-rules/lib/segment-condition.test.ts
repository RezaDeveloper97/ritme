import { describe, expect, it } from 'vitest';

import type { FactDef } from '@/features/rule-conditions';
import { fromStored, toStored, emptyConditions } from '@/features/rule-conditions';

import { addSegment, removeSegment, segmentIds, withoutSegments, withSegments } from './segment-condition';

const FACTS: FactDef[] = [
  { name: 'segment', type: 'set', enum: null, unit: null, group: 'segment', description: {}, member_safe: false, pending: null },
  { name: 'mode', type: 'enum', enum: ['cycle', 'ttc'], unit: null, group: 'account', description: {}, member_safe: false, pending: null },
];

const stored = [{ op: 'and' as const, items: [{ fact: 'mode', op: 'eq', value: 'ttc' }, { fact: 'segment', op: 'contains', value: '7' }] }];

describe('segment condition', () => {
  it('reads segment ids and splits them from the builder rows', () => {
    const f = fromStored(stored, []);
    expect(segmentIds(f)).toEqual(['7']);
    expect(withoutSegments(f).allOf.map((c) => c.fact)).toEqual(['mode']);
  });

  it('adds, dedupes and clears; the saved rule round-trips', () => {
    const base = withoutSegments(fromStored(stored, []));
    const two = withSegments(base, addSegment(addSegment(['7'], '9'), '9'));
    const out = toStored(two, FACTS);
    expect(out.conditions[0]?.items).toEqual([
      { fact: 'segment', op: 'contains', value: '7' },
      { fact: 'segment', op: 'contains', value: '9' },
      { fact: 'mode', op: 'eq', value: 'ttc' },
    ]);
    expect(segmentIds(fromStored(out.conditions, out.exceptions))).toEqual(['7', '9']);
    const cleared = withSegments(base, removeSegment(['7'], '7'));
    expect(segmentIds(cleared)).toEqual([]);
    expect(toStored(cleared, FACTS).conditions[0]?.items).toHaveLength(1);
  });

  it('an empty builder with no segment stays a single blank row; a lone segment drops it', () => {
    expect(withSegments(emptyConditions(), []).allOf).toHaveLength(1);
    expect(withSegments(emptyConditions(), ['3']).allOf).toEqual([{ fact: 'segment', op: 'contains', value: '3' }]);
  });
});
