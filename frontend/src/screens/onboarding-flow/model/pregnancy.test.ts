import { describe, expect, it } from 'vitest';

import { addDays, fromApiDate, toApiDate } from '@/shared/lib/date';

import { gestationalDays, pregnancyOnboardingBody, pregnancySummary, type PregnancyAnswers } from './pregnancy';

const today = fromApiDate('2026-10-01');
const base: PregnancyAnswers = {
  basis: 'lmp', lmp: null, scanDate: null, scanWeeks: null, scanDays: 0, dueDate: null, weeks: null, days: 0,
};

describe('pregnancy dating', () => {
  it('dates from the last period', () => {
    const a = { ...base, lmp: addDays(today, -62) };
    expect(gestationalDays(a, today)).toBe(62);
    expect(pregnancySummary(a, today)).toMatchObject({ weeks: 8, days: 6 });
    expect(toApiDate(pregnancySummary(a, today)!.due)).toBe(toApiDate(addDays(today, 218)));
    expect(pregnancyOnboardingBody(a, today)).toEqual({ age_source: 'lmp', lmp_date: toApiDate(addDays(today, -62)) });
  });

  it('dates from an ultrasound (age at scan + days since)', () => {
    const a = { ...base, basis: 'ultrasound' as const, scanDate: addDays(today, -10), scanWeeks: 7, scanDays: 3 };
    expect(gestationalDays(a, today)).toBe(62);
    expect(pregnancyOnboardingBody(a, today)).toMatchObject({ age_source: 'ultrasound', ultrasound_weeks: 7, ultrasound_days: 3 });
    expect(gestationalDays({ ...a, scanWeeks: null }, today)).toBeNull();
  });

  it('turns a due date into the LMP it implies', () => {
    const a = { ...base, basis: 'due' as const, dueDate: addDays(today, 218) };
    expect(gestationalDays(a, today)).toBe(62);
    expect(pregnancyOnboardingBody(a, today)).toEqual({ age_source: 'lmp', lmp_date: toApiDate(addDays(today, -62)) });
  });

  it('takes the current week as a manual entry', () => {
    const a = { ...base, basis: 'week' as const, weeks: 12, days: 2 };
    expect(gestationalDays(a, today)).toBe(86);
    expect(pregnancyOnboardingBody(a, today)).toEqual({ age_source: 'manual', manual_weeks: 12, manual_days: 2 });
  });

  it('rejects impossible ages', () => {
    expect(gestationalDays({ ...base, lmp: addDays(today, 3) }, today)).toBeNull();
    expect(gestationalDays({ ...base, lmp: addDays(today, -400) }, today)).toBeNull();
    expect(pregnancyOnboardingBody(base, today)).toBeNull();
  });
});
