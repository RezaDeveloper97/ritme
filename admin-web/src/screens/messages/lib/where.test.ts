import { describe, expect, it } from 'vitest';

import fa from '../../../../messages/fa.json';
import { AREAS, areaOf, describeItemKey, groupByArea } from './where';

describe('message where-map', () => {
  it('every mapped group and area has its Persian words', () => {
    const groups = Object.keys(fa.smartMessages.groups);
    for (const g of groups) {
      expect(fa.smartMessages.where, g).toHaveProperty(g);
      expect(areaOf(g), g).not.toBeNull();
    }
    for (const a of AREAS) expect(fa.smartMessages.areas, a).toHaveProperty(a);
  });

  it('buckets groups by area in a stable order, unknown last', () => {
    expect(groupByArea(['companion_tip', 'new_thing', 'cycle_base_ttc', 'pattern'])).toEqual([
      { area: 'cycleHome', groups: ['cycle_base_ttc', 'pattern'] },
      { area: 'companion', groups: ['companion_tip'] },
      { area: null, groups: ['new_thing'] },
    ]);
  });

  it('names composite and numbered keys', () => {
    const known = (k: string) => ['luteal', 'menstrual', 'pain'].includes(k);
    expect(describeItemKey('nutrition_trimester', '2', known)).toEqual({ kind: 'trimester', n: 2 });
    expect(describeItemKey('pregnancy_week_tip', '12', known)).toEqual({ kind: 'week', n: 12 });
    expect(describeItemKey('companion_tip', 'luteal_tip_2', known)).toEqual({ kind: 'phaseTip', phase: 'luteal', n: 2 });
    expect(describeItemKey('companion_tip', 'menstrual_note', known)).toEqual({ kind: 'phaseNote', phase: 'menstrual' });
    expect(describeItemKey('cycle_override', 'pain', known)).toEqual({ kind: 'key', key: 'pain' });
    expect(describeItemKey('pattern', 'zzz', known)).toEqual({ kind: 'raw' });
  });
});
