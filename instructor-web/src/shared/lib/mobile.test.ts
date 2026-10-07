import { describe, expect, it } from 'vitest';

import { formatMobile, normalizeMobile, toPersianDigits } from './mobile';

describe('normalizeMobile', () => {
  it.each([
    ['09123456789', '09123456789'],
    ['0912 345 6789', '09123456789'],
    ['۰۹۱۲۳۴۵۶۷۸۹', '09123456789'],
    ['٠٩١٢٣٤٥٦٧٨٩', '09123456789'],
    ['+989123456789', '09123456789'],
    ['00989123456789', '09123456789'],
    ['989123456789', '09123456789'],
    ['9123456789', '09123456789'],
  ])('%s → %s', (input, out) => {
    expect(normalizeMobile(input)).toBe(out);
  });

  it.each(['', '0912345678', '021345678901', '08123456789', 'abc'])('rejects %s', (input) => {
    expect(normalizeMobile(input)).toBeNull();
  });
});

describe('formatting', () => {
  it('groups a mobile 4-3-4', () => {
    expect(formatMobile('09123456789')).toBe('0912 345 6789');
  });
  it('writes Persian digits', () => {
    expect(toPersianDigits('0:42')).toBe('۰:۴۲');
  });
});
