import { describe, expect, it } from 'vitest';

import { setPrepItemDone, toAppointmentBody } from './body';

describe('toAppointmentBody', () => {
  it('serialises the form in the README meta shape', () => {
    expect(
      toAppointmentBody({
        kind: 'in_person',
        withWhom: 'دکتر احمدی',
        specialty: 'زنان و زایمان',
        topic: 'ultrasound',
        title: '',
        date: '2026-09-30',
        time: '10:30',
        location: 'مطب',
        remindBefore: '1d',
        addToCalendar: true,
        notes: null,
        prep: [
          { id: 'p1', text: ' دفترچه بیمه ', done: false },
          { id: 'p2', text: '   ', done: false },
        ],
      }),
    ).toEqual({
      kind: 'in_person',
      with: 'دکتر احمدی',
      specialty: 'زنان و زایمان',
      topic: 'ultrasound',
      title: null,
      scheduled_at: '2026-09-30 10:30:00',
      location: 'مطب',
      remind_before: '1d',
      add_to_calendar: true,
      notes: null,
      prep: [{ id: 'p1', text: 'دفترچه بیمه', done: false }],
    });
  });

  it('stays partial and needs both date and time for scheduled_at', () => {
    expect(toAppointmentBody({ isActive: false })).toEqual({ is_active: false });
    expect(toAppointmentBody({ date: '2026-09-30' })).toEqual({});
  });
});

describe('setPrepItemDone', () => {
  it('flips only the target item, immutably', () => {
    const prep = [
      { id: 'p1', text: 'a', done: false },
      { id: 'p2', text: 'b', done: false },
    ];
    const next = setPrepItemDone(prep, 'p2', true);
    expect(next.map((i) => i.done)).toEqual([false, true]);
    expect(prep[1].done).toBe(false);
  });
});

describe('toAppointmentBody — pregnancy fields', () => {
  it('maps care_item_key, stage and result_note', () => {
    expect(toAppointmentBody({ careItemKey: 'nt_scan', stage: 'result', resultNote: ' ok ' })).toEqual({
      care_item_key: 'nt_scan',
      stage: 'result',
      result_note: 'ok',
    });
    expect(toAppointmentBody({ careItemKey: '', resultNote: '' })).toEqual({ care_item_key: null, result_note: null });
    expect(toAppointmentBody({ stage: 'done' })).toEqual({ stage: 'done' });
  });
});
