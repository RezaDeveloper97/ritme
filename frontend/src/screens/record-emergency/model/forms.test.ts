import { describe, expect, it } from 'vitest';

import { emergencyLinkUrl, isValidLast4, isValidPhone, normalizeLast4, normalizePhone } from './forms';

describe('emergency card forms', () => {
  it('normalizes Persian digits and separators in a phone', () => {
    expect(normalizePhone('۰۹۱۲ ۱۱۱-۲۲۳۳')).toBe('09121112233');
    expect(normalizePhone('+98 912 111 2233')).toBe('+989121112233');
  });

  it('accepts 4–15 digits', () => {
    expect(isValidPhone('115')).toBe(false);
    expect(isValidPhone('۱۱۵۰')).toBe(true);
    expect(isValidPhone('1234567890123456')).toBe(false);
  });

  it('takes exactly four insurance digits', () => {
    expect(normalizeLast4('۴۸۲۱')).toBe('4821');
    expect(isValidLast4('۴۸۲۱')).toBe(true);
    expect(isValidLast4('482')).toBe(false);
    expect(isValidLast4('48210')).toBe(false);
  });

  it('builds the public link', () => {
    expect(emergencyLinkUrl('https://web.ritme.app/', '/fa/emergency/abc')).toBe('https://web.ritme.app/fa/emergency/abc');
  });
});
