import { describe, expect, it } from 'vitest';

import type { CareToday, TodayDose } from '@/entities/care-reminder';

import { cardStatus, doseRowState, slotClock, slotPeriod } from './dose-row';

const dose = (over: Partial<TodayDose> = {}): TodayDose => ({
  reminderId: 12,
  title: 'فولیک اسید ۴۰۰ میکروگرم',
  form: 'tablet',
  slot: '08:00',
  taken: false,
  ...over,
});

const today = (over: Partial<CareToday> = {}): CareToday => ({
  date: '2026-09-23',
  doses: [],
  takenCount: 0,
  total: 0,
  nextAppointment: null,
  ...over,
});

describe('doseRowState', () => {
  it('maps a taken dose to a filled check whose tap unticks', () => {
    const row = doseRowState(dose({ taken: true }));
    expect(row.taken).toBe(true);
    expect(row.nextTaken).toBe(false);
  });

  it('maps an untaken dose to an empty ring whose tap ticks', () => {
    const row = doseRowState(dose({ taken: false }));
    expect(row.taken).toBe(false);
    expect(row.nextTaken).toBe(true);
  });

  it('keys each slot of one medication separately', () => {
    expect(doseRowState(dose({ slot: '08:00' })).key).not.toBe(
      doseRowState(dose({ slot: '20:00' })).key,
    );
  });

  it('picks the icon and tint from the form', () => {
    expect(doseRowState(dose({ form: 'tablet' }))).toMatchObject({ icon: 'tablet', tone: 'brand' });
    expect(doseRowState(dose({ form: 'capsule' }))).toMatchObject({ icon: 'capsule', tone: 'teal' });
    expect(doseRowState(dose({ form: 'drops' }))).toMatchObject({ icon: 'drop', tone: 'brand' });
  });

  it('shows the slot on a 12-hour clock with its part of day', () => {
    expect(doseRowState(dose({ slot: '08:00' }))).toMatchObject({ clock: '8:00', period: 'morning' });
    expect(doseRowState(dose({ slot: '21:00' }))).toMatchObject({ clock: '9:00', period: 'night' });
  });
});

describe('slot helpers', () => {
  it.each([
    ['05:00', 'morning'],
    ['11:59', 'morning'],
    ['12:00', 'noon'],
    ['14:30', 'noon'],
    ['15:00', 'evening'],
    ['18:59', 'evening'],
    ['19:00', 'night'],
    ['00:30', 'night'],
    ['04:59', 'night'],
  ])('%s is %s', (slot, period) => {
    expect(slotPeriod(slot)).toBe(period);
  });

  it('formats midnight and noon as 12', () => {
    expect(slotClock('00:15')).toBe('12:15');
    expect(slotClock('12:00')).toBe('12:00');
    expect(slotClock('13:05')).toBe('1:05');
  });
});

describe('cardStatus', () => {
  it('shows a skeleton while loading', () => {
    expect(cardStatus({ isLoading: true, isError: false, data: undefined })).toBe('loading');
  });

  it('hides the card on error', () => {
    expect(cardStatus({ isLoading: false, isError: true, data: undefined })).toBe('hidden');
  });

  it('is empty with no doses and no appointment', () => {
    expect(cardStatus({ isLoading: false, isError: false, data: today() })).toBe('empty');
  });

  it('is ready with only an appointment or only doses', () => {
    const appointment = {
      id: 40,
      kind: 'in_person' as const,
      title: 'سونوگرافی NT',
      withWhom: 'دکتر احمدی',
      scheduledAt: '2026-09-30 10:30:00',
      daysUntil: 7,
      location: null,
      remindBefore: '1d' as const,
      isActive: true,
    };
    expect(
      cardStatus({ isLoading: false, isError: false, data: today({ nextAppointment: appointment }) }),
    ).toBe('ready');
    expect(
      cardStatus({ isLoading: false, isError: false, data: today({ doses: [dose()], total: 1 }) }),
    ).toBe('ready');
  });
});
