import { describe, expect, it } from 'vitest';

import { bookingWhen } from './booking';

describe('bookingWhen', () => {
  it('reads the Tehran wall-clock date and time from the ISO string', () => {
    const w = bookingWhen({ startsAt: '2026-10-15T18:00:00+03:30' }, new Date(2026, 9, 7));
    expect(w.time).toBe('18:00');
    expect(w.daysAway).toBe(8);
    expect(w.date.getDate()).toBe(15);
  });

  it('never goes negative', () => {
    expect(bookingWhen({ startsAt: '2026-10-01T09:30:00+03:30' }, new Date(2026, 9, 7)).daysAway).toBe(0);
  });
});
