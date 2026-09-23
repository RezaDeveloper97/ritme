import { describe, expect, it } from 'vitest';

import { nextSearch, parseListParams, toApiQuery } from './list-params';

describe('list params', () => {
  it('parses with defaults and clamps per_page', () => {
    const p = parseListParams(new URLSearchParams('page=3&per_page=500&q=%20sara%20'), { status: 'all' });
    expect(p).toEqual({ page: 3, perPage: 100, q: 'sara', filters: { status: 'all' } });
    expect(parseListParams(new URLSearchParams('page=-1')).page).toBe(1);
  });

  it('builds the API query', () => {
    const p = parseListParams(new URLSearchParams('status=blocked'), { status: 'all' });
    expect(toApiQuery(p)).toEqual({ page: 1, per_page: 20, status: 'blocked' });
  });

  it('resets the page on filter change and drops defaults', () => {
    const cur = new URLSearchParams('page=4&q=a');
    expect(nextSearch(cur, { status: 'blocked' }, { status: 'all' })).toBe('?q=a&status=blocked');
    expect(nextSearch(cur, { status: 'all' }, { status: 'all' })).toBe('?q=a');
    expect(nextSearch(cur, { page: 5 })).toBe('?page=5&q=a');
    expect(nextSearch(new URLSearchParams('page=2'), { page: 1 })).toBe('');
  });
});
