import { describe, expect, it } from 'vitest';

import { stringListSchema, translationsSchema } from './resource';

describe('resource schemas', () => {
  it('translationsSchema normalises null, packed [] and non-strings to an object of strings', () => {
    expect(translationsSchema.parse(null)).toEqual({});
    expect(translationsSchema.parse([])).toEqual({});
    expect(translationsSchema.parse({ fa: 'x', en: null, ar: 3 })).toEqual({ fa: 'x', ar: '3' });
  });

  it('stringListSchema keeps only strings', () => {
    expect(stringListSchema.parse(null)).toEqual([]);
    expect(stringListSchema.parse(['a', 1, 'b'])).toEqual(['a', 'b']);
  });
});
