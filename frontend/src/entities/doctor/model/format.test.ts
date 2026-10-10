import { describe, expect, it } from 'vitest';

import { dayOffset, formatToman, startsAtParam, toman, videoWindowOpen } from './format';

describe('doctor format', () => {
  it('shows integer rials as toman', () => {
    expect(toman(2_500_000)).toBe(250_000);
    expect(formatToman(2_500_000, 'en')).toBe('250,000');
    expect(formatToman(2_500_000, 'fa')).toBe('۲۵۰٬۰۰۰');
  });
  it('offsets days from today', () => {
    const now = new Date(2026, 9, 10, 15);
    expect(dayOffset('2026-10-10', now)).toBe(0);
    expect(dayOffset('2026-10-11', now)).toBe(1);
  });
  it('builds the starts_at wall clock', () => expect(startsAtParam('2026-10-14', '18:00')).toBe('2026-10-14 18:00'));
  it('opens the video window 10 min before until the end', () => {
    const o = '2026-10-14T17:50:00+03:30';
    const c = '2026-10-14T18:20:00+03:30';
    expect(videoWindowOpen(o, c, Date.parse('2026-10-14T17:49:00+03:30'))).toBe(false);
    expect(videoWindowOpen(o, c, Date.parse('2026-10-14T18:00:00+03:30'))).toBe(true);
    expect(videoWindowOpen(o, c, Date.parse('2026-10-14T18:20:00+03:30'))).toBe(false);
  });
});
