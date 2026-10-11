import { describe, expect, it } from 'vitest';

import { bookable, minPrice, monthsLabelParts, neshanHref, priceParts, safeWebsite, slotDayOffset, slotParts } from './place';

describe('place helpers', () => {
  it('formats prices', () => {
    expect(priceParts(320000)).toEqual({ unit: 'thousand', value: 320 });
    expect(priceParts(2400000)).toEqual({ unit: 'million', value: 2.4 });
  });
  it('parses slot timestamps', () => {
    expect(slotParts('2026-10-12T17:00:00+03:30')).toEqual({ day: '2026-10-12', time: '17:00' });
    expect(slotParts('2026-10-12 09:30:00')).toEqual({ day: '2026-10-12', time: '09:30' });
    expect(slotParts('nope')).toBeNull();
    expect(slotDayOffset('2026-10-12', new Date(2026, 9, 11))).toBe(1);
  });
  it('labels months', () => {
    expect(monthsLabelParts(6)).toEqual({ unit: 'month', n: 6 });
    expect(monthsLabelParts(48)).toEqual({ unit: 'year', n: 4 });
  });
  it('allows only http(s) websites', () => {
    expect(safeWebsite('https://a.ir/x')).toBe('https://a.ir/x');
    expect(safeWebsite('javascript:alert(1)')).toBeNull();
    expect(safeWebsite(null)).toBeNull();
  });
  it('picks min price, bookable slots, neshan link', () => {
    expect(minPrice([{ price_toman: 0 }, { price_toman: 650000 }, { price_toman: 320000 }, { price_toman: null }] as never)).toBe(320000);
    expect(bookable({ is_full: false, seats_left: 2 } as never)).toBe(true);
    expect(bookable({ is_full: true, seats_left: 0 } as never)).toBe(false);
    expect(neshanHref(35.7, 51.4)).toContain('35.7,51.4');
  });
});
