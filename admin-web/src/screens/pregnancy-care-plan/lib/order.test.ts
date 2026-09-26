import { describe, expect, it } from 'vitest';

import { cleanTranslations, moveItem } from './order';

describe('care plan helpers', () => {
  it('moves an item', () => {
    expect(moveItem([1, 2, 3], 0, 2)).toEqual([2, 3, 1]);
    expect(moveItem([1, 2], 0, 5)).toEqual([1, 2]);
  });
  it('cleans translations', () => {
    expect(cleanTranslations({ fa: ' ', en: 'x ' })).toEqual({ en: 'x' });
    expect(cleanTranslations({ fa: '' })).toBeNull();
  });
});
