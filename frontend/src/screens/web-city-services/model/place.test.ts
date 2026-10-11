import { describe, expect, it } from 'vitest';

import { bookable, minPrice, monthsLabelParts, priceParts, safeHttp, slotParts, telHref } from './place';

describe('web place helpers (CB-DIR-11)', () => {
  it('formats declared prices in thousands under a million and millions above', () => {
    expect(priceParts(320_000)).toEqual({ unit: 'thousand', value: 320 });
    expect(priceParts(2_400_000)).toEqual({ unit: 'million', value: 2.4 });
  });

  it('splits an API slot stamp into day and time', () => {
    expect(slotParts('2026-10-12T17:00:00+03:30')).toEqual({ day: '2026-10-12', time: '17:00' });
    expect(slotParts('2026-10-12 09:30:00')).toEqual({ day: '2026-10-12', time: '09:30' });
    expect(slotParts('nope')).toBeNull();
  });

  it('labels ages in months, whole years from two', () => {
    expect(monthsLabelParts(6)).toEqual({ unit: 'month', n: 6 });
    expect(monthsLabelParts(24)).toEqual({ unit: 'year', n: 2 });
    expect(monthsLabelParts(30)).toEqual({ unit: 'month', n: 30 });
  });

  it('picks the cheapest positive price', () => {
    const svc = (price: number | null) => ({ price_toman: price }) as never;
    expect(minPrice([svc(350_000), svc(null), svc(320_000), svc(0)])).toBe(320_000);
    expect(minPrice([svc(null)])).toBeNull();
  });

  it('lets only http(s) links and a clean tel href out', () => {
    expect(safeHttp('https://example.com/x')).toBe('https://example.com/x');
    expect(safeHttp('javascript:alert(1)')).toBeNull();
    expect(safeHttp(null)).toBeNull();
    expect(telHref('+98 21 1234-5678')).toBe('tel:+982112345678');
  });

  it('treats a full or zero-seat slot as not bookable', () => {
    expect(bookable({ is_full: false, seats_left: 2 } as never)).toBe(true);
    expect(bookable({ is_full: true, seats_left: 2 } as never)).toBe(false);
    expect(bookable({ is_full: false, seats_left: 0 } as never)).toBe(false);
  });
});
