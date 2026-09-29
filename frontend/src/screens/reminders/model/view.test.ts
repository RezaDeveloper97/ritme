import { describe, expect, it } from 'vitest';

import type { Appointment, Medication } from '@/entities/care-reminder';

import {
  appointmentRowState,
  doseCardState,
  medicationRowState,
  medicationSchedule,
  nextTab,
  parseTab,
  sectionStatus,
  slotClock,
  slotPeriod,
  sortMedications,
  tabHref,
  visibleSections,
} from './view';

const med = (over: Partial<Medication> = {}): Medication => ({
  id: 12,
  title: 'فولیک اسید',
  subtitle: '۴۰۰ میکروگرم',
  dose: '400',
  unit: 'mcg',
  form: 'tablet',
  times: ['08:00'],
  weekdays: [0, 1, 2, 3, 4, 5, 6],
  amount: 1,
  duration: 'ongoing',
  notify: true,
  startsOn: '2026-09-23',
  endsOn: null,
  isActive: true,
  notes: null,
  ...over,
});

const appt = (over: Partial<Appointment> = {}): Appointment => ({
  id: 40,
  title: 'سونوگرافی NT',
  subtitle: null,
  kind: 'in_person',
  withWhom: 'دکتر احمدی',
  specialty: null,
  topic: 'ultrasound',
  location: 'مطب',
  remindBefore: '1d',
  addToCalendar: false,
  prep: [],
  status: 'scheduled',
  scheduledAt: '2026-09-30 10:30:00',
  isActive: true,
  notes: null,
  ...over,
});

describe('tabs', () => {
  it('parses ?tab= and falls back to all', () => {
    expect(parseTab('medications')).toBe('medications');
    expect(parseTab('appointments')).toBe('appointments');
    expect(parseTab(undefined)).toBe('all');
    expect(parseTab('bogus')).toBe('all');
  });

  it('keeps the default tab out of the URL', () => {
    expect(tabHref('all')).toBe('/reminders');
    expect(tabHref('appointments')).toBe('/reminders?tab=appointments');
  });

  it('filters sections per tab', () => {
    expect(visibleSections('all')).toEqual({ today: true, medications: true, appointments: true });
    expect(visibleSections('medications')).toEqual({ today: true, medications: true, appointments: false });
    expect(visibleSections('appointments')).toEqual({ today: false, medications: false, appointments: true });
  });

  it('moves with the arrows in reading direction and wraps', () => {
    expect(nextTab('all', 'ArrowLeft', 'rtl')).toBe('medications');
    expect(nextTab('all', 'ArrowRight', 'ltr')).toBe('medications');
    expect(nextTab('all', 'ArrowRight', 'rtl')).toBe('appointments');
    expect(nextTab('appointments', 'ArrowLeft', 'rtl')).toBe('all');
    expect(nextTab('medications', 'Home', 'ltr')).toBe('all');
    expect(nextTab('medications', 'End', 'ltr')).toBe('appointments');
    expect(nextTab('medications', 'Enter', 'ltr')).toBeNull();
  });
});

describe('slots', () => {
  it('formats a 12-hour clock and a day period', () => {
    expect(slotClock('08:00')).toBe('8:00');
    expect(slotClock('21:30')).toBe('9:30');
    expect(slotClock('00:05')).toBe('12:05');
    expect(slotPeriod('08:00')).toBe('morning');
    expect(slotPeriod('13:00')).toBe('noon');
    expect(slotPeriod('17:00')).toBe('evening');
    expect(slotPeriod('21:00')).toBe('night');
  });

  it('maps a dose to its card', () => {
    const card = doseCardState({ reminderId: 3, title: 'ویتامین D', form: 'capsule', slot: '21:00', taken: false });
    expect(card).toMatchObject({ key: '3-21:00', clock: '9:00', period: 'night', nextTaken: true });
  });
});

describe('medicationRowState', () => {
  it('shows every day, the dose and a brand tile for tablets', () => {
    const row = medicationRowState(med());
    expect(row.schedule).toEqual({ kind: 'everyDay' });
    expect(row.dose).toBe('۴۰۰ میکروگرم');
    expect(row.tone).toBe('brand');
    expect(row.icon).toBe('tablet');
    expect(row.slots).toEqual([{ clock: '8:00', period: 'morning' }]);
  });

  it('counts a weekday subset and tints capsules teal', () => {
    const row = medicationRowState(med({ form: 'capsule', weekdays: [0, 1, 4], subtitle: ' ' }));
    expect(row.schedule).toEqual({ kind: 'daysPerWeek', count: 3 });
    expect(row.tone).toBe('teal');
    expect(row.dose).toBeNull();
  });
});

describe('medicationSchedule', () => {
  it('reads alternating days as every other day', () => {
    expect(medicationSchedule([0, 2, 4, 6])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([1, 3, 5])).toEqual({ kind: 'everyOtherDay' });
    expect(medicationSchedule([0, 2])).toEqual({ kind: 'daysPerWeek', count: 2 });
    expect(medicationSchedule([0, 1, 2, 3, 4, 5, 6])).toEqual({ kind: 'everyDay' });
    expect(medicationSchedule([])).toEqual({ kind: 'everyDay' });
  });
});

describe('sortMedications', () => {
  it('lists active reminders first, each by its first slot', () => {
    const list = sortMedications([
      med({ id: 3, times: ['13:00'], isActive: false }),
      med({ id: 2, times: ['21:00'] }),
      med({ id: 1, times: ['08:00', '20:00'] }),
    ]);
    expect(list.map((m) => m.id)).toEqual([1, 2, 3]);
  });
});

describe('appointmentRowState', () => {
  it('splits the wall-clock time; in person puts who in the title and the place in the meta', () => {
    expect(appointmentRowState(appt())).toMatchObject({
      tone: 'amber',
      date: '2026-09-30',
      time: '10:30',
      titleWith: 'دکتر احمدی',
      detail: 'مطب',
    });
  });

  it('shows who in the meta for a phone or online consultation', () => {
    expect(appointmentRowState(appt({ kind: 'phone' }))).toMatchObject({ titleWith: null, detail: 'دکتر احمدی' });
  });

  it('tints consultations teal and falls back to the place', () => {
    const row = appointmentRowState(appt({ kind: 'online', withWhom: null, scheduledAt: null }));
    expect(row).toMatchObject({ tone: 'teal', date: null, time: null, detail: 'مطب' });
  });
});

describe('sectionStatus', () => {
  it('orders error, loading, empty, ready', () => {
    expect(sectionStatus({ isLoading: false, isError: true, data: undefined })).toBe('error');
    expect(sectionStatus({ isLoading: true, isError: false, data: undefined })).toBe('loading');
    expect(sectionStatus({ isLoading: false, isError: false, data: [] })).toBe('empty');
    expect(sectionStatus({ isLoading: false, isError: false, data: [1] })).toBe('ready');
  });
});
