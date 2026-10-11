import { describe, expect, it } from 'vitest';

import { formatToman } from './price';

describe('formatToman', () => {
  it('compacts thousands and millions', () => {
    expect(formatToman(485000, 'en')).toEqual({ unit: 'thousand', n: '485' });
    expect(formatToman(9800000, 'en')).toEqual({ unit: 'million', n: '9.8' });
    expect(formatToman(9800000, 'fa')).toEqual({ unit: 'million', n: '۹٫۸' });
    expect(formatToman(500, 'en')).toEqual({ unit: 'plain', n: '500' });
  });
});
