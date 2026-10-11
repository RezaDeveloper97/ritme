import { describe, expect, it } from 'vitest';

import { heroKey, isDefaultList, NO_FILTERS, storeProductHref } from './store';

describe('web shop model', () => {
  it('detects the untouched list state', () => {
    expect(isDefaultList(NO_FILTERS, 'popular', 1)).toBe(true);
    expect(isDefaultList(NO_FILTERS, 'rating', 1)).toBe(false);
    expect(isDefaultList(NO_FILTERS, 'popular', 2)).toBe(false);
    expect(isDefaultList({ ...NO_FILTERS, size: ['0-3'] }, 'popular', 1)).toBe(false);
    expect(isDefaultList({ ...NO_FILTERS, price_min: '100' }, 'popular', 1)).toBe(false);
  });
  it('maps hero copy and links', () => {
    expect(heroKey('beauty')).toBe('beauty');
    expect(heroKey('anything')).toBe('other');
    expect(storeProductHref(7)).toBe('/store/product/7');
  });
});
