import { describe, expect, it } from 'vitest';

import { toMedicationBody } from './body';

describe('toMedicationBody', () => {
  it('serialises the form in the README meta shape', () => {
    expect(
      toMedicationBody({
        title: ' فولیک اسید ',
        dose: '400',
        unit: 'mcg',
        form: 'tablet',
        times: ['20:00', '08:00', '08:00'],
        weekdays: [6, 0, 3],
        amount: 1,
        startsOn: '2026-09-21',
        duration: 'pregnancy_end',
        endsOn: '2027-01-01',
        notify: true,
        notes: '  ',
      }),
    ).toEqual({
      title: 'فولیک اسید',
      dose: '400',
      unit: 'mcg',
      form: 'tablet',
      times: ['08:00', '20:00'],
      weekdays: [0, 3, 6],
      amount: 1,
      starts_on: '2026-09-21',
      duration: 'pregnancy_end',
      ends_on: null,
      notify: true,
      notes: null,
    });
  });

  it('keeps ends_on for until_date and stays partial for a PUT', () => {
    expect(toMedicationBody({ duration: 'until_date', endsOn: '2026-12-01' })).toEqual({
      duration: 'until_date',
      ends_on: '2026-12-01',
    });
    expect(toMedicationBody({ isActive: false })).toEqual({ is_active: false });
  });
});
