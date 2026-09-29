import { describe, expect, it } from 'vitest';

import { bookPrefill, heroRelative, markDoneSheetArg, monthsSince } from './view';

describe('checkup detail view', () => {
  it('counts whole months since the last visit', () => {
    const now = new Date(2026, 8, 26);
    expect(monthsSince(null, now)).toBeNull();
    expect(monthsSince('2026-03-20', now)).toBe(6);
    expect(monthsSince('2026-09-30', now)).toBe(0);
  });
  it('builds the mark-done sheet arg', () => {
    expect(markDoneSheetArg(4)).toBe('4');
    expect(markDoneSheetArg(4, 9)).toBe('4-9');
  });
  it('hero: months past the due date, only while overdue', () => {
    const now = new Date(2026, 8, 26);
    expect(heroRelative({ status: 'due', lastDoneOn: null, nextDueOn: null }, now)).toEqual({ kind: 'never' });
    expect(heroRelative({ status: 'overdue', lastDoneOn: '2022-04-10', nextDueOn: '2026-03-20' }, now)).toEqual({
      kind: 'done',
      lastDoneOn: '2022-04-10',
      overdueMonths: 6,
    });
    expect(heroRelative({ status: 'up_to_date', lastDoneOn: '2026-02-01', nextDueOn: '2027-02-01' }, now)).toEqual({
      kind: 'done',
      lastDoneOn: '2026-02-01',
      overdueMonths: null,
    });
  });
});

describe('bookPrefill (B-4)', () => {
  it('hands the checkup title to the appointment form, like the home card', () => {
    expect(bookPrefill({ title: 'پاپ‌اسمیر / HPV' })).toEqual({ title: 'پاپ‌اسمیر / HPV' });
  });
});
