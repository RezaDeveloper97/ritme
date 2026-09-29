import { describe, expect, it } from 'vitest';

import { formatLocaleInteger, parseLocaleInteger } from './locale-number';

describe('locale-number', () => {
  it('shows the locale digits', () => {
    expect(formatLocaleInteger(8, 'fa')).toBe('۸');
    expect(formatLocaleInteger(12, 'fa')).toBe('۱۲');
    expect(formatLocaleInteger(12, 'en')).toBe('12');
    expect(formatLocaleInteger(0, 'fa')).toBe('۰');
    expect(formatLocaleInteger(undefined, 'fa')).toBe('');
  });

  it('reads Persian, Arabic-Indic and ASCII digits', () => {
    expect(parseLocaleInteger('۱۲')).toBe(12);
    expect(parseLocaleInteger('٣')).toBe(3);
    expect(parseLocaleInteger('40')).toBe(40);
    expect(parseLocaleInteger('۴a2')).toBe(42);
    expect(parseLocaleInteger('')).toBeUndefined();
    expect(parseLocaleInteger('-')).toBeUndefined();
    expect(parseLocaleInteger('1234', 2)).toBe(12);
  });
});
