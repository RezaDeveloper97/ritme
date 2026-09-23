import { describe, expect, it } from 'vitest';

import { formatDate, formatDateTime, tehranHour } from './date';

describe('date layer', () => {
  it('shows Jalali with Persian digits for fa', () => {
    // 2026-09-23 = 1 Mehr 1405.
    const out = formatDateTime('2026-09-23T13:05:00+03:30', 'fa');
    expect(out).toContain('۱۴۰۵');
    expect(out).toContain('۱۳:۰۵');
    expect(formatDate('2026-09-23', 'fa')).toContain('مهر');
  });

  it('shows Gregorian for any other locale, in Tehran time', () => {
    expect(formatDateTime('2026-09-23T13:05:00+03:30', 'en')).toBe('09/23/2026 13:05');
    // 22:00 UTC is already the next day in Tehran.
    expect(formatDate('2026-09-23T22:00:00Z', 'en')).toBe('Sep 24, 2026');
  });

  it('returns an empty string for null or garbage', () => {
    expect(formatDateTime(null, 'fa')).toBe('');
    expect(formatDate('not a date', 'en')).toBe('');
  });

  it('reads the Tehran hour', () => {
    expect(tehranHour(new Date('2026-09-23T20:30:00Z'))).toBe(0);
  });
});
