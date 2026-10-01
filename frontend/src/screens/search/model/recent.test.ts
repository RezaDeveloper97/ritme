import { beforeEach, describe, expect, it } from 'vitest';

import { addRecentSearch, clearRecentSearches, readRecentSearches, RECENT_SEARCH_LIMIT } from './recent';

const store = new Map<string, string>();
Object.defineProperty(globalThis, 'window', {
  value: {
    localStorage: {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => void store.set(k, v),
      removeItem: (k: string) => void store.delete(k),
    },
  },
  configurable: true,
});

describe('recent searches', () => {
  beforeEach(() => store.clear());

  it('keeps the newest first, deduplicated and capped', () => {
    addRecentSearch('1', 'درد');
    addRecentSearch('1', 'قرص آهن');
    expect(addRecentSearch('1', ' درد ')).toEqual(['درد', 'قرص آهن']);
    for (let i = 0; i < 10; i += 1) addRecentSearch('1', `q${i}`);
    expect(readRecentSearches('1')).toHaveLength(RECENT_SEARCH_LIMIT);
  });

  it("never shows another account's list", () => {
    addRecentSearch('1', 'درد');
    expect(readRecentSearches('2')).toEqual([]);
    expect(readRecentSearches('1')).toEqual([]);
  });

  it('clears, and survives garbage', () => {
    addRecentSearch('1', 'درد');
    clearRecentSearches();
    expect(readRecentSearches('1')).toEqual([]);
    store.set('ritme_search_recent', '{not json');
    expect(readRecentSearches('1')).toEqual([]);
  });
});
