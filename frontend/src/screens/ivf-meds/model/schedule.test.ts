import { describe, expect, it } from 'vitest';

import type { IvfInventory, IvfSites } from '@/entities/ivf';

import { chosenSite, rotationNote, runsOutLabel, siteForLog, sortForInventory, stockState } from './schedule';

const SITES: IvfSites = {
  codes: ['abdomen_upper_right', 'abdomen_upper_left', 'thigh_right'],
  last: { site: 'abdomen_upper_right', date: '2026-09-23', slot: '08:00' },
  suggested: 'abdomen_upper_left',
};

const inv = (over: Partial<IvfInventory> = {}): IvfInventory => ({
  stockUnits: 3,
  stockUnit: 'pen',
  dosesPerUnit: 1,
  countedAt: '2026-09-23 10:00:00',
  dosesLeft: 3,
  unitsLeft: 3,
  daysLeft: 3,
  runsOutOn: '2026-09-26',
  low: true,
  ...over,
});

describe('site rotation', () => {
  it('pre-selects the suggested (least recently used) site, then her pick', () => {
    expect(chosenSite(SITES, null)).toBe('abdomen_upper_left');
    expect(chosenSite(SITES, 'thigh_right')).toBe('thigh_right');
    expect(chosenSite(SITES, 'arm_left')).toBe('abdomen_upper_left');
    expect(chosenSite({ ...SITES, suggested: null }, null)).toBe('abdomen_upper_right');
    expect(chosenSite({ codes: [], last: null, suggested: null }, null)).toBeNull();
  });

  it('describes last vs today', () => {
    expect(rotationNote(SITES, 'abdomen_upper_left')).toEqual({
      last: 'abdomen_upper_right',
      today: { site: 'abdomen_upper_left', kind: 'suggested' },
    });
    expect(rotationNote({ ...SITES, last: null }, 'thigh_right').today).toEqual({ site: 'thigh_right', kind: 'picked' });
  });

  it('sends a site only with an injected medicine', () => {
    expect(siteForLog({ route: 'subcutaneous' }, 'thigh_right')).toBe('thigh_right');
    expect(siteForLog({ route: 'intramuscular' }, 'thigh_right')).toBe('thigh_right');
    expect(siteForLog({ route: 'vaginal' }, 'thigh_right')).toBeNull();
    expect(siteForLog({ route: 'oral' }, 'thigh_right')).toBeNull();
  });
});

describe('inventory', () => {
  it('maps low / ok / none', () => {
    expect(stockState(inv())).toBe('low');
    expect(stockState(inv({ low: false }))).toBe('ok');
    expect(stockState(null)).toBe('none');
  });

  it('names the run-out day: weekday within a week, else a date', () => {
    expect(runsOutLabel(inv(), '2026-09-23')).toEqual({ kind: 'weekday', on: '2026-09-26' });
    expect(runsOutLabel(inv({ runsOutOn: '2026-10-05' }), '2026-09-23')).toEqual({ kind: 'date', on: '2026-10-05' });
    expect(runsOutLabel(inv({ daysLeft: null }), '2026-09-23')).toBeNull();
    expect(runsOutLabel(null, '2026-09-23')).toBeNull();
  });

  it('lists low stock first, untracked last', () => {
    const meds = [
      { name: 'B', inventory: null },
      { name: 'C', inventory: inv({ low: false }) },
      { name: 'A', inventory: inv() },
    ];
    expect(sortForInventory(meds).map((m) => m.name)).toEqual(['A', 'C', 'B']);
  });
});
