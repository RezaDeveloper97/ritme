import { describe, expect, it } from 'vitest';

import { keepReachable, subphasesFor } from './subphases';

const subs = [{ value: 'menstrual' }, { value: 'follicular_early' }, { value: 'follicular_late' }, { value: 'luteal_mid' }];
const phaseOf = { menstrual: 'menstrual', follicular_early: 'follicular', follicular_late: 'follicular', luteal_mid: 'luteal' };

describe('subphase picker rules', () => {
  it('shows every sub-phase without a phase, only the phase’s own otherwise', () => {
    expect(subphasesFor(subs, phaseOf, '').visible).toHaveLength(4);
    const f = subphasesFor(subs, phaseOf, 'follicular');
    expect(f.visible.map((s) => s.value)).toEqual(['follicular_early', 'follicular_late']);
    expect(f.narrowable).toBe(true);
    expect(subphasesFor(subs, phaseOf, 'menstrual').narrowable).toBe(false);
  });

  it('drops selections outside the phase, and all of them when it cannot be narrowed', () => {
    const f = subphasesFor(subs, phaseOf, 'follicular');
    expect(keepReachable(['follicular_late', 'luteal_mid'], f.visible, f.narrowable)).toEqual(['follicular_late']);
    const m = subphasesFor(subs, phaseOf, 'menstrual');
    expect(keepReachable(['menstrual'], m.visible, m.narrowable)).toEqual([]);
  });
});
