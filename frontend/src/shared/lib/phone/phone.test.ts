import { describe, expect, it } from 'vitest';

import { isValidMobile, normalizeMobile, toAsciiDigits, toPersianDigits } from './index';

describe('toAsciiDigits', () => {
  it('converts Persian and Arabic-Indic digits to ASCII', () => {
    expect(toAsciiDigits('۰۹۱۲')).toBe('0912');
    expect(toAsciiDigits('٠٩١٢')).toBe('0912');
    expect(toAsciiDigits('09-abc')).toBe('09-abc');
  });
});

describe('toPersianDigits', () => {
  it('renders ASCII digits as Persian', () => {
    expect(toPersianDigits('0912')).toBe('۰۹۱۲');
    expect(toPersianDigits(59)).toBe('۵۹');
  });
});

describe('normalizeMobile', () => {
  it('strips separators and Persian digits to a bare local number', () => {
    expect(normalizeMobile('۰۹۱۲ ۳۴۵ ۶۷۸۹')).toBe('09123456789');
    expect(normalizeMobile('0912-345-6789')).toBe('09123456789');
  });

  it('collapses +98 / 0098 / 98 prefixes to local form', () => {
    expect(normalizeMobile('+989123456789')).toBe('09123456789');
    expect(normalizeMobile('00989123456789')).toBe('09123456789');
    expect(normalizeMobile('989123456789')).toBe('09123456789');
  });

  it('caps length at 11 digits', () => {
    expect(normalizeMobile('091234567890000')).toHaveLength(11);
  });
});

describe('isValidMobile', () => {
  it('accepts a well-formed Iranian mobile number', () => {
    expect(isValidMobile('09123456789')).toBe(true);
    expect(isValidMobile('+98 912 345 6789')).toBe(true);
  });

  it('rejects malformed numbers', () => {
    expect(isValidMobile('0912345678')).toBe(false); // too short
    expect(isValidMobile('08123456789')).toBe(false); // wrong prefix
    expect(isValidMobile('')).toBe(false);
  });
});
