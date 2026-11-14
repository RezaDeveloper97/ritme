import { describe, expect, it } from 'vitest';

import { toMedicationBody, withForUser } from './body';

describe('toMedicationBody', () => {
  it('serialises the dose-block shape and never the legacy fields', () => {
    const body = toMedicationBody({
      title: ' فولیک اسید ',
      startsOn: '2026-09-21',
      frequency: { every: 2, unit: 'hour' },
      doses: [
        { time: '08:00', amount: 1, unit: 'pill', meal: 'after' },
        { time: '20:00', amount: 2, unit: 'drop', meal: null },
      ],
      duration: 'pregnancy_end',
      endsOn: '2027-01-01',
      notify: true,
      notes: '  ',
    });
    expect(body).toEqual({
      title: 'فولیک اسید',
      starts_on: '2026-09-21',
      frequency: { every: 2, unit: 'hour' },
      doses: [
        { time: '08:00', amount: 1, unit: 'pill', meal: 'after' },
        { time: '20:00', amount: 2, unit: 'drop', meal: null },
      ],
      duration: 'pregnancy_end',
      ends_on: null,
      notify: true,
      notes: null,
    });
    for (const k of ['times', 'weekdays', 'form', 'dose', 'unit', 'amount']) expect(body).not.toHaveProperty(k);
  });

  it('keeps ends_on for until_date and stays partial for a PUT', () => {
    expect(toMedicationBody({ duration: 'until_date', endsOn: '2026-12-01' })).toEqual({
      duration: 'until_date',
      ends_on: '2026-12-01',
    });
    expect(toMedicationBody({ isActive: false })).toEqual({ is_active: false });
  });
});

describe('withForUser (B-N4-06)', () => {
  it('adds for_user_id only for an owner', () => {
    const body = toMedicationBody({ title: 'آهن' });
    expect(withForUser(body, 42)).toEqual({ title: 'آهن', for_user_id: 42 });
    expect(withForUser(body, null)).toEqual({ title: 'آهن' });
    expect(withForUser(body, undefined)).toBe(body);
  });
});
