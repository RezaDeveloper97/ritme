import { describe, expect, it } from 'vitest';

import { ageMonthsOf, EMPTY_FILTERS, filterCount, slotWhen, toDirectoryParams } from './filters';

const catalog = [
  { code: 'a', name: '6-18', meta: { min_months: 6, max_months: 18 } },
  { code: 'b', name: 'open', meta: { min_months: 24 } },
  { code: 'c', name: 'none', meta: null },
];

describe('directory filters', () => {
  it('maps a catalog age range to the middle of its span', () => {
    expect(ageMonthsOf('a', catalog)).toBe(12);
    expect(ageMonthsOf('b', catalog)).toBe(24);
    expect(ageMonthsOf('c', catalog)).toBeNull();
    expect(ageMonthsOf(null, catalog)).toBeNull();
  });

  it('counts filters but not the sort', () => {
    expect(filterCount(EMPTY_FILTERS)).toBe(0);
    expect(filterCount({ age: 'a', openNow: true, amenities: ['x', 'y'], sort: 'rating' })).toBe(4);
  });

  it('falls back from distance sort when no position was given, and only sends a position the user shared', () => {
    const f = { ...EMPTY_FILTERS, sort: 'distance' as const };
    expect(toDirectoryParams(f, catalog, null)).toMatchObject({ sort: 'rank' });
    expect(toDirectoryParams(f, catalog, null)).not.toHaveProperty('lat');
    expect(toDirectoryParams(f, catalog, { lat: 35.7, lng: 51.4 })).toMatchObject({ sort: 'distance', lat: 35.7, lng: 51.4 });
  });

  it('labels the next slot relative to today', () => {
    const now = new Date(2026, 9, 11, 9, 0);
    expect(slotWhen(new Date(2026, 9, 11, 17, 0).toISOString(), now)?.day).toBe('today');
    expect(slotWhen(new Date(2026, 9, 12, 10, 30).toISOString(), now)?.day).toBe('tomorrow');
    expect(slotWhen(new Date(2026, 9, 15, 10, 30).toISOString(), now)?.day).toBe('date');
    expect(slotWhen(new Date(2026, 9, 10, 10, 30).toISOString(), now)).toBeNull();
    expect(slotWhen(null, now)).toBeNull();
    expect(slotWhen('nope', now)).toBeNull();
  });
});
