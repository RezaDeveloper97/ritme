import { describe, expect, it } from 'vitest';

import type { Appointment } from '@/entities/care-reminder';
import { buildIcs } from '@/shared/lib/ics';

import {
  appointmentIcsEvent,
  daysUntil,
  mapsHref,
  pregnancyWeekAt,
  splitScheduled,
  weekdayOf,
} from './detail';

const appt: Appointment = {
  id: 9,
  title: 'سونوگرافی NT',
  subtitle: null,
  kind: 'in_person',
  withWhom: 'دکتر رضایی',
  specialty: null,
  topic: 'ultrasound',
  location: 'تهران، ولیعصر',
  remindBefore: '1d',
  addToCalendar: true,
  prep: [],
  status: 'scheduled',
  scheduledAt: '2026-10-05 10:30:00',
  isActive: true,
  notes: null,
};

describe('appointment detail model', () => {
  it('splits the Tehran wall clock', () => {
    expect(splitScheduled('2026-10-05 10:30:00')).toEqual({ date: '2026-10-05', time: '10:30' });
    expect(splitScheduled(null)).toBeNull();
  });

  it('counts days until the visit', () => {
    expect(daysUntil('2026-10-05', '2026-09-26')).toBe(9);
    expect(daysUntil('2026-09-26', '2026-09-26')).toBe(0);
  });

  it('computes the pregnancy week on the appointment date', () => {
    expect(pregnancyWeekAt(83, 9)).toBe(14); // 12w6d today → 14w1d
    expect(pregnancyWeekAt(null, 3)).toBeNull();
    expect(pregnancyWeekAt(400, 0)).toBeNull();
  });

  it('only offers directions for an in-person address', () => {
    expect(mapsHref(appt)).toContain(encodeURIComponent('تهران، ولیعصر'));
    expect(mapsHref({ ...appt, kind: 'online' })).toBeNull();
    expect(mapsHref({ ...appt, location: null })).toBeNull();
  });

  it('builds the calendar event with alarm = remind_before in Tehran time', () => {
    const event = appointmentIcsEvent(appt, appt.title, null)!;
    const ics = buildIcs({ ...event, stamp: new Date(0) });
    expect(ics).toContain('DTSTART;TZID=Asia/Tehran:20261005T103000');
    expect(ics).toContain('TRIGGER:-P1D');
    expect(buildIcs(appointmentIcsEvent({ ...appt, isActive: false }, 'x', null)!)).not.toContain('VALARM');
    expect(appointmentIcsEvent({ ...appt, scheduledAt: null }, 'x', null)).toBeNull();
  });
});

describe('reminderMoment', () => {
  it('subtracts remind_before across a day boundary', async () => {
    const { reminderMoment } = await import('./detail');
    expect(reminderMoment({ date: '2026-10-01', time: '01:30' }, '3h')).toEqual({ date: '2026-09-30', time: '22:30' });
    expect(reminderMoment({ date: '2026-10-05', time: '10:30' }, '1d')).toEqual({ date: '2026-10-04', time: '10:30' });
  });
});

describe('weekdayOf', () => {
  it('names the weekday of a Y-m-d date', () => {
    expect(weekdayOf('2026-09-30')).toBe('wed'); // ۸ مهر ۱۴۰۵
    expect(weekdayOf('2026-10-03')).toBe('sat');
    expect(weekdayOf('garbled')).toBeNull();
  });
});
