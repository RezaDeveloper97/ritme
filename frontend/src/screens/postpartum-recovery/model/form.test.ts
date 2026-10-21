import { describe, expect, it } from 'vitest';

import { diffForm, formFrom, locationMissing, roundTemperature, toggleIn } from './form';

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

  it('red flags: sends the temperature and the yes/no flags that changed, null clears', () => {
    expect(empty.temperatureC).toBeNull();
    expect(empty.breathless).toBeNull();
    expect(diffForm(empty, { ...empty, temperatureC: 38, breathless: true })).toEqual({ temperatureC: 38, breathless: true });
    const saved = { ...empty, temperatureC: 38.2, severePain: true };
    expect(diffForm(saved, { ...saved, temperatureC: null, severePain: false })).toEqual({ temperatureC: null, severePain: false });
    expect(diffForm(saved, saved)).toEqual({});
  });

  it('reads the saved red flags from the API day', () => {
    const f = formFrom({
      date: '2026-10-21',
      lochiaAmount: null,
      lochiaColor: null,
      painLevel: null,
      painLocations: [],
      breasts: null,
      feedsCount: null,
      sleepHours: null,
      temperatureC: 38.1,
      padSoakedWithinHour: true,
      severePain: null,
      breathless: false,
      postpartumVisit: null,
      alerts: [],
    });
    expect(f).toMatchObject({ temperatureC: 38.1, padSoakedWithinHour: true, severePain: null, breathless: false });
  });

  it('keeps the temperature at one decimal inside the bounds', () => {
    expect(roundTemperature(37.1 + 0.1, 34, 43)).toBe(37.2);
    expect(roundTemperature(33.9, 34, 43)).toBe(34);
    expect(roundTemperature(43.1, 34, 43)).toBe(43);
  });
});
