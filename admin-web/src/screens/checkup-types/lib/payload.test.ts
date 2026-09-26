import { describe, expect, it } from 'vitest';

import { cleanTranslations, moveItem } from './payload';

describe('moveItem', () => {
  it('moves up and down', () => {
    expect(moveItem([1, 2, 3], 2, 0)).toEqual([3, 1, 2]);
    expect(moveItem([1, 2, 3], 0, 1)).toEqual([2, 1, 3]);
  });
  it('ignores out-of-range moves', () => {
    expect(moveItem([1, 2], 1, 2)).toEqual([1, 2]);
  });
});

describe('cleanTranslations', () => {
  it('drops blanks and returns null when empty', () => {
    expect(cleanTranslations({ fa: ' x ', en: '' })).toEqual({ fa: 'x' });
    expect(cleanTranslations({ fa: '  ' })).toBeNull();
  });
});
