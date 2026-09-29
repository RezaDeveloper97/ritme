import { describe, expect, it } from 'vitest';

import { toLatinDigits, toLocaleDigits, toNumberText } from './number';

describe('number field text', () => {
  it('reads Persian, Arabic-Indic and ASCII digits alike', () => {
    expect(toLatinDigits('۱۲۳')).toBe('123');
    expect(toLatinDigits('١٢٣')).toBe('123');
    expect(toNumberText('۱2٣')).toBe('123');
    expect(toNumberText('12')).toBe('12');
  });

  it('keeps one leading minus and one decimal point, drops the rest', () => {
    expect(toNumberText('-۵')).toBe('-5');
    expect(toNumberText('−5')).toBe('-5');
    expect(toNumberText('۱٫۵')).toBe('1.5');
    expect(toNumberText('1.2.3')).toBe('1.23');
    expect(toNumberText('1-2')).toBe('12');
    expect(toNumberText('abc')).toBe('');
    expect(toNumberText('')).toBe('');
  });

  it("shows the stored text in the locale's digits", () => {
    expect(toLocaleDigits('12', 'fa')).toBe('۱۲');
    expect(toLocaleDigits('-1.5', 'fa')).toBe('-۱.۵');
    expect(toLocaleDigits('12', 'en')).toBe('12');
    expect(toLocaleDigits('', 'fa')).toBe('');
  });
});
