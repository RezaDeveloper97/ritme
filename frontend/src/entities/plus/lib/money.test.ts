import { describe, expect, it } from 'vitest';

import { formatToman, formatTomanThousands, rialsToToman } from './money';

describe('rialsToToman', () => {
  it('divides by ten and rounds', () => {
    expect(rialsToToman(2370000)).toBe(237000);
    expect(rialsToToman(2346300)).toBe(234630);
    expect(rialsToToman(5)).toBe(1);
    expect(rialsToToman(0)).toBe(0);
  });
});

describe('formatToman', () => {
  it('groups thousands with Persian digits and the Arabic separator in fa', () => {
    expect(formatToman(2370000, 'fa')).toBe('۲۳۷٬۰۰۰');
    expect(formatToman(990000, 'fa')).toBe('۹۹٬۰۰۰');
  });

  it('uses Latin digits and commas elsewhere', () => {
    expect(formatToman(2346300, 'en')).toBe('234,630');
    expect(formatToman(39000000, 'en')).toBe('3,900,000');
    expect(formatToman(9000, 'en')).toBe('900');
  });

  it('prints the magnitude only (the caller adds the minus sign of a discount)', () => {
    expect(formatToman(-237000, 'en')).toBe('23,700');
  });
});

describe('formatTomanThousands', () => {
  it('shows thousands of toman with one optional decimal', () => {
    expect(formatTomanThousands(2370000, 'fa')).toBe('۲۳۷');
    expect(formatTomanThousands(1185000, 'fa')).toBe('۱۱۸٫۵');
    expect(formatTomanThousands(395000, 'en')).toBe('39.5');
    expect(formatTomanThousands(123450000, 'en')).toBe('12,345');
  });
});
