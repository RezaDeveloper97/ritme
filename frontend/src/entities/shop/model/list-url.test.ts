import { describe, expect, it } from 'vitest';

import { DEFAULT_LIST_STATE, parseShopListState, shopListParams, shopListSearch } from './list-url';

const parse = (s: string) => {
  const p = new URLSearchParams(s);
  return parseShopListState((k) => p.get(k));
};

describe('shop list URL state', () => {
  it('default state is an empty query string', () => {
    expect(shopListSearch(DEFAULT_LIST_STATE)).toBe('');
    expect(parse('')).toEqual(DEFAULT_LIST_STATE);
  });

  it('round-trips filters, sort, q and page', () => {
    const s = parse('q=pad&size=S,M&color=red&skin_type=oily&price_min=1000&price_max=5000&sort=price_asc&page=3');
    expect(s.filters.size).toEqual(['S', 'M']);
    expect(s.page).toBe(3);
    expect(parse(shopListSearch(s).slice(1))).toEqual(s);
  });

  it('ignores malformed values', () => {
    const s = parse('sort=bogus&page=-2&price_min=ab12');
    expect(s.sort).toBe('popular');
    expect(s.page).toBe(1);
    expect(s.filters.price_min).toBe('12');
  });

  it('maps to API params', () => {
    const p = shopListParams('pads', parse('size=S,M&price_max=900'));
    expect(p).toMatchObject({ category: 'pads', size: 'S,M', price_max: 900, sort: 'popular', page: 1 });
    expect(p.q).toBeUndefined();
  });
});
