import { describe, expect, it } from 'vitest';

import { diffForm, formFrom, locationMissing, toggleIn } from './form';

const empty = formFrom(undefined);

describe('recovery form', () => {
  it('sends only what changed', () => {
    expect(diffForm(empty, empty)).toEqual({});
    expect(diffForm(empty, { ...empty, lochiaAmount: 'light', feedsCount: 8 })).toEqual({ lochiaAmount: 'light', feedsCount: 8 });
  });

  it('pain level and places travel together; no pain clears the places', () => {
    const saved = { ...empty, painLevel: 'mild' as const, painLocations: ['stitches' as const] };
    expect(diffForm(saved, { ...saved, painLocations: ['stitches', 'back'] })).toEqual({
      painLevel: 'mild',
      painLocations: ['stitches', 'back'],
    });
    expect(diffForm(saved, { ...saved, painLevel: 'none' })).toEqual({ painLevel: 'none', painLocations: [] });
  });

  it('a real pain level needs a place', () => {
    expect(locationMissing({ ...empty, painLevel: 'moderate' })).toBe(true);
    expect(locationMissing({ ...empty, painLevel: 'none' })).toBe(false);
  });

  it('breasts: normal is [] and differs from not logged', () => {
    expect(diffForm(empty, { ...empty, breasts: [] })).toEqual({ breasts: [] });
    expect(toggleIn(['redness'], 'redness')).toEqual([]);
  });
});
