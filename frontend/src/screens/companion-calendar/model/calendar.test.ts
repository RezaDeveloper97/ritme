import { describe, expect, it } from 'vitest';

import { appointmentsByDay, monthWindow, periodDaySet, pickPartner } from './calendar';

describe('monthWindow', () => {
  it('spans a Gregorian month', () => {
    expect(monthWindow(2026, 2, 'en')).toEqual({ from: '2026-02-01', to: '2026-02-28' });
  });
  it('spans a Jalali month for fa', () => {
    // Aban 1405 = 2026-10-23 … 2026-11-21.
    expect(monthWindow(1405, 8, 'fa')).toEqual({ from: '2026-10-23', to: '2026-11-21' });
  });
});

describe('appointmentsByDay', () => {
  it('groups by day, sorts by time and drops cancelled or undated ones', () => {
    const map = appointmentsByDay([
      { id: 1, title: 'b', scheduledAt: '2026-11-20 17:00:00', status: 'scheduled' },
      { id: 2, title: 'a', scheduledAt: '2026-11-20 09:30:00', status: 'scheduled' },
      { id: 3, title: 'x', scheduledAt: '2026-11-21 10:00:00', status: 'cancelled' },
      { id: 4, title: 'y', scheduledAt: '', status: 'scheduled' },
    ]);
    expect(map.get('2026-11-20')?.map((a) => a.id)).toEqual([2, 1]);
    expect(map.has('2026-11-21')).toBe(false);
    expect(map.size).toBe(1);
  });
});

describe('pickPartner / periodDaySet', () => {
  const partners = [
    { linkId: 1, partnerName: 'Sara', periodDays: ['2026-11-25'] },
    { linkId: 2, partnerName: null, periodDays: null },
  ];
  it('falls back to the first partner', () => {
    expect(pickPartner(partners, 2)?.linkId).toBe(2);
    expect(pickPartner(partners, 9)?.linkId).toBe(1);
    expect(pickPartner([], null)).toBeNull();
  });
  it('reads an unshared list as empty', () => {
    expect(periodDaySet(partners[0]!).has('2026-11-25')).toBe(true);
    expect(periodDaySet(partners[1]!).size).toBe(0);
  });
});
