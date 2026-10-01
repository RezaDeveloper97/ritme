import { describe, expect, it } from 'vitest';

import { LIFE_MODES } from '@/entities/user';

import { MODE_CARDS, needsConfirm, planSwitch } from './modes';

describe('MODE_CARDS', () => {
  it('lists the six life modes in artboard order', () => {
    expect(MODE_CARDS.map((c) => c.key)).toEqual([...LIFE_MODES]);
  });
});

describe('planSwitch', () => {
  it('does nothing for the current mode', () => {
    expect(planSwitch('teen', 'teen')).toEqual({ kind: 'none' });
    expect(planSwitch('pregnancy', 'pregnancy')).toEqual({ kind: 'none' });
  });

  it('goes through the pregnancy setup to enter pregnancy', () => {
    expect(planSwitch('cycle', 'pregnancy')).toEqual({ kind: 'enterPregnancy' });
    expect(planSwitch('menopause', 'pregnancy')).toEqual({ kind: 'enterPregnancy' });
  });

  it('deactivates the pregnancy profile before leaving pregnancy', () => {
    expect(planSwitch('pregnancy', 'cycle')).toEqual({ kind: 'leavePregnancy', target: 'cycle' });
    expect(planSwitch('pregnancy', 'postpartum')).toEqual({ kind: 'leavePregnancy', target: 'postpartum' });
  });

  it('stores every other change directly', () => {
    expect(planSwitch('cycle', 'ttc')).toEqual({ kind: 'store', target: 'ttc' });
    expect(planSwitch('ttc', 'menopause')).toEqual({ kind: 'store', target: 'menopause' });
  });

  it('confirms only pregnancy changes', () => {
    expect(needsConfirm(planSwitch('cycle', 'pregnancy'))).toBe(true);
    expect(needsConfirm(planSwitch('pregnancy', 'teen'))).toBe(true);
    expect(needsConfirm(planSwitch('cycle', 'teen'))).toBe(false);
  });
});
