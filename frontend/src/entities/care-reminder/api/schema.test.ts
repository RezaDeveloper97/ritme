import { describe, expect, it } from 'vitest';

import { applyIntake } from '../model/intake';
import { careKeys } from './keys';
import {
  appointmentSchema,
  careEnumsSchema,
  careTodaySchema,
  medicationListSchema,
  medicationSchema,
} from './schema';

/*
 * Boundary contract for `/api/v1/care/*` — fixtures are the examples in
 * docs/care-reminders/README.md. The frontend was written before the Go
 * endpoints, so the parsers must also survive shapes the README leaves open
 * (nested vs flattened `meta`) and enum values this bundle has never seen.
 */

const medicationMeta = {
  v: 1,
  dose: '400',
  unit: 'mcg',
  form: 'tablet',
  times: ['20:00', '08:00'],
  weekdays: [0, 1, 2, 3, 4, 5, 6],
  amount: 1,
  duration: 'pregnancy_end',
  notify: true,
};

const medicationRow = {
  id: 12,
  type: 'medication',
  title: 'فولیک اسید',
  subtitle: '۴۰۰ میکروگرم',
  recurrence: 'daily',
  recurrence_time: '08:00:00',
  starts_on: '2026-09-21',
  ends_on: '2027-05-18',
  is_active: true,
  notes: null,
};

const appointmentMeta = {
  v: 1,
  kind: 'in_person',
  with: 'دکتر احمدی',
  specialty: 'زنان و زایمان',
  topic: 'ultrasound',
  location: '…',
  remind_before: '1d',
  add_to_calendar: true,
  prep: [{ id: 'p1', text: 'دفترچه بیمه', done: false }],
  status: 'scheduled',
};

const appointmentRow = {
  id: 40,
  type: 'appointment',
  title: 'سونوگرافی NT هفته ۱۱',
  subtitle: 'دکتر احمدی · زنان و زایمان',
  scheduled_at: '2026-09-30 10:30:00',
  recurrence: 'none',
  is_active: true,
  notes: null,
};

const todayFixture = {
  date: '2026-09-23',
  doses: [
    { reminder_id: 12, title: 'فولیک اسید ۴۰۰ میکروگرم', form: 'tablet', slot: '08:00', taken: true },
  ],
  taken_count: 1,
  total: 2,
  next_appointment: {
    id: 40,
    kind: 'in_person',
    title: 'سونوگرافی NT',
    with: 'دکتر احمدی',
    scheduled_at: '2026-09-30 10:30:00',
    days_until: 8,
    location: 'مطب',
    remind_before: '1d',
  },
};

describe('medicationSchema', () => {
  it('reads a row with nested meta (README data model)', () => {
    const m = medicationSchema.parse({ ...medicationRow, meta: medicationMeta });
    expect(m).toEqual({
      id: 12,
      title: 'فولیک اسید',
      subtitle: '۴۰۰ میکروگرم',
      dose: '400',
      unit: 'mcg',
      form: 'tablet',
      times: ['08:00', '20:00'],
      weekdays: [0, 1, 2, 3, 4, 5, 6],
      amount: 1,
      duration: 'pregnancy_end',
      notify: true,
      startsOn: '2026-09-21',
      endsOn: '2027-05-18',
      isActive: true,
      notes: null,
    });
  });

  it('reads the same row flattened, and meta as a JSON string', () => {
    const flat = medicationSchema.parse({ ...medicationRow, ...medicationMeta });
    const stringMeta = medicationSchema.parse({ ...medicationRow, meta: JSON.stringify(medicationMeta) });
    const nested = medicationSchema.parse({ ...medicationRow, meta: medicationMeta });
    expect(flat).toEqual(nested);
    expect(stringMeta).toEqual(nested);
  });

  it('falls back safely on unknown enum values and junk', () => {
    const m = medicationSchema.parse({
      ...medicationRow,
      id: '12',
      meta: {
        ...medicationMeta,
        form: 'patch',
        duration: 'forever',
        weekdays: [0, 9, '3', 3, 'x'],
        amount: 'lots',
        times: ['08:00:00', '08:00'],
      },
    });
    expect(m.id).toBe(12);
    expect(m.form).toBe('tablet');
    expect(m.duration).toBe('ongoing');
    expect(m.weekdays).toEqual([0, 3]);
    expect(m.amount).toBe(1);
    expect(m.times).toEqual(['08:00']);
  });

  it('defaults a legacy row that has no meta at all', () => {
    const m = medicationSchema.parse({ id: 3, title: 'آهن' });
    expect(m).toMatchObject({
      form: 'tablet',
      times: [],
      weekdays: [0, 1, 2, 3, 4, 5, 6],
      amount: 1,
      duration: 'ongoing',
      notify: true,
      isActive: true,
      dose: '',
      unit: '',
    });
  });

  it('parses a list', () => {
    expect(medicationListSchema.parse([{ ...medicationRow, meta: medicationMeta }])).toHaveLength(1);
  });
});

describe('appointmentSchema', () => {
  it('reads a row with nested meta (README data model)', () => {
    const a = appointmentSchema.parse({ ...appointmentRow, meta: appointmentMeta });
    expect(a).toEqual({
      id: 40,
      title: 'سونوگرافی NT هفته ۱۱',
      subtitle: 'دکتر احمدی · زنان و زایمان',
      kind: 'in_person',
      withWhom: 'دکتر احمدی',
      specialty: 'زنان و زایمان',
      topic: 'ultrasound',
      location: '…',
      remindBefore: '1d',
      addToCalendar: true,
      prep: [{ id: 'p1', text: 'دفترچه بیمه', done: false }],
      status: 'scheduled',
      scheduledAt: '2026-09-30 10:30:00',
      isActive: true,
      notes: null,
    });
  });

  it('falls back safely on unknown enum values', () => {
    const a = appointmentSchema.parse({
      ...appointmentRow,
      meta: { ...appointmentMeta, kind: 'house_call', topic: 'dental', remind_before: '1w', status: 'moved' },
    });
    expect(a.kind).toBe('in_person');
    expect(a.topic).toBe('other');
    expect(a.remindBefore).toBe('1d');
    expect(a.status).toBe('scheduled');
  });

  it('keeps a cancelled appointment cancelled and drops a malformed prep list', () => {
    const a = appointmentSchema.parse({
      ...appointmentRow,
      is_active: false,
      meta: { ...appointmentMeta, status: 'cancelled', prep: [{ nope: true }] },
    });
    expect(a.status).toBe('cancelled');
    expect(a.isActive).toBe(false);
    expect(a.prep).toEqual([]);
  });
});

describe('careTodaySchema', () => {
  it('reads the README example', () => {
    expect(careTodaySchema.parse(todayFixture)).toEqual({
      date: '2026-09-23',
      doses: [{ reminderId: 12, title: 'فولیک اسید ۴۰۰ میکروگرم', form: 'tablet', slot: '08:00', taken: true }],
      takenCount: 1,
      total: 2,
      nextAppointment: {
        id: 40,
        kind: 'in_person',
        title: 'سونوگرافی NT',
        withWhom: 'دکتر احمدی',
        scheduledAt: '2026-09-30 10:30:00',
        daysUntil: 8,
        location: 'مطب',
        remindBefore: '1d',
      },
    });
  });

  it('sorts doses by slot, derives missing counts, maps unknown form', () => {
    const t = careTodaySchema.parse({
      date: '2026-09-23',
      doses: [
        { reminder_id: 2, title: 'D', form: 'gummy', slot: '21:00', taken: false },
        { reminder_id: 1, title: 'F', form: 'capsule', slot: '08:00:00', taken: true },
      ],
      next_appointment: null,
    });
    expect(t.doses.map((d) => d.slot)).toEqual(['08:00', '21:00']);
    expect(t.doses[1].form).toBe('tablet');
    expect(t.takenCount).toBe(1);
    expect(t.total).toBe(2);
    expect(t.nextAppointment).toBeNull();
  });

  it('hides a malformed next appointment instead of failing the card', () => {
    const t = careTodaySchema.parse({ ...todayFixture, next_appointment: { id: 'x' } });
    expect(t.nextAppointment).toBeNull();
    expect(t.doses).toHaveLength(1);
  });
});

describe('careEnumsSchema', () => {
  it('reads option lists, drops unknown values, accepts a value→label map', () => {
    const e = careEnumsSchema.parse({
      forms: [
        { value: 'tablet', label: 'قرص' },
        { value: 'patch', label: 'چسب' },
      ],
      units: { mcg: 'میکروگرم', custom_unit: 'پیمانه' },
      kinds: [{ value: 'phone', label: 'مشاوره تلفنی' }],
      topics: [{ value: 'lab' }],
      remind_before: [{ value: '1h', label: '۱ ساعت قبل' }],
    });
    expect(e.forms).toEqual([{ value: 'tablet', label: 'قرص' }]);
    expect(e.units).toEqual([
      { value: 'mcg', label: 'میکروگرم' },
      { value: 'custom_unit', label: 'پیمانه' },
    ]);
    expect(e.kinds).toEqual([{ value: 'phone', label: 'مشاوره تلفنی' }]);
    expect(e.topics).toEqual([{ value: 'lab', label: 'lab' }]);
    expect(e.remindBefore).toEqual([{ value: '1h', label: '۱ ساعت قبل' }]);
    expect(e.durations).toEqual([]);
  });
});

describe('applyIntake', () => {
  const today = careTodaySchema.parse({
    ...todayFixture,
    doses: [
      ...todayFixture.doses,
      { reminder_id: 13, title: 'ویتامین D', form: 'capsule', slot: '21:00', taken: false },
    ],
  });

  it('ticks a dose and bumps the count', () => {
    const next = applyIntake(today, { reminderId: 13, slot: '21:00', taken: true });
    expect(next.doses[1].taken).toBe(true);
    expect(next.takenCount).toBe(2);
    expect(today.doses[1].taken).toBe(false); // immutable
  });

  it('unticks a dose', () => {
    const next = applyIntake(today, { reminderId: 12, slot: '08:00', taken: false });
    expect(next.doses[0].taken).toBe(false);
    expect(next.takenCount).toBe(0);
  });

  it('is a no-op for an unknown dose or an unchanged state', () => {
    expect(applyIntake(today, { reminderId: 99, slot: '08:00', taken: true })).toBe(today);
    expect(applyIntake(today, { reminderId: 12, slot: '08:00', taken: true })).toBe(today);
  });
});

describe('careKeys', () => {
  it('nests every variant under its prefix', () => {
    expect(careKeys.today('2026-09-23').slice(0, 2)).toEqual([...careKeys.todayAll()]);
    expect(careKeys.medication(12).slice(0, 2)).toEqual([...careKeys.medicationsAll()]);
    expect(careKeys.appointments('past').slice(0, 2)).toEqual([...careKeys.appointmentsAll()]);
    expect(careKeys.today()).toEqual(['care', 'today', 'current']);
  });
});
