import { describe, expect, it } from 'vitest';

import { ACTION_CATALOGUE, homeActions, orderedCategories } from './actions';

const ids = (s: Parameters<typeof orderedCategories>[0]) => orderedCategories(s).map((c) => c.id);

describe('orderedCategories', () => {
  it('keeps the default order without settings', () => {
    expect(ids(null)).toEqual(['calm', 'food', 'help', 'gift', 'time']);
    expect(ids({ goals: [], actionPrefs: [] })).toEqual(['calm', 'food', 'help', 'gift', 'time']);
  });

  it('puts the owner preferences first, in her order, then the goal category', () => {
    expect(ids({ goals: ['low_energy'], actionPrefs: ['time', 'food'] })).toEqual(['time', 'food', 'help', 'calm', 'gift']);
  });

  it('lifts the category of every goal, in the order she picked them', () => {
    expect(ids({ goals: ['quality_time', 'cycle', 'low_energy'], actionPrefs: [] })).toEqual([
      'time',
      'help',
      'calm',
      'food',
      'gift',
    ]);
  });

  it('lets the goal lift its category when no preference is set', () => {
    expect(ids({ goals: ['quality_time'], actionPrefs: [] })[0]).toBe('time');
  });

  it('never duplicates a category', () => {
    expect(ids({ goals: ['support'], actionPrefs: ['calm', 'calm'] })).toEqual(['calm', 'food', 'help', 'gift', 'time']);
  });
});

describe('homeActions', () => {
  it('takes the first card of the top three categories', () => {
    const picked = homeActions({ goals: [], actionPrefs: ['gift'] });
    expect(picked.map((a) => `${a.category}:${a.key}`)).toEqual(['gift:smallGesture', 'calm:listen', 'food:warmDrink']);
  });

  it('every category has two or three cards', () => {
    for (const c of ACTION_CATALOGUE) {
      expect(c.cards.length).toBeGreaterThanOrEqual(2);
      expect(c.cards.length).toBeLessThanOrEqual(3);
    }
  });
});
