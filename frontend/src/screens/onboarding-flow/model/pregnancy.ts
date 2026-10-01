import type { OnboardingInput } from '@/entities/pregnancy';
import { addDays, diffInDays, toApiDate } from '@/shared/lib/date';

/**
 * Dating basis of nbl_Onb_Preg: last period, ultrasound, due date or the
 * current week. Pure (dates in, numbers out) and unit-tested.
 */
export type PregnancyBasis = 'lmp' | 'ultrasound' | 'due' | 'week';

export const PREGNANCY_BASES: readonly PregnancyBasis[] = ['lmp', 'ultrasound', 'due', 'week'];

/** A full-term pregnancy, in days (40 weeks). */
export const TERM_DAYS = 280;
/** The longest gestational age the API accepts (42 weeks). */
export const MAX_GA_DAYS = 42 * 7 + 6;

export interface PregnancyAnswers {
  basis: PregnancyBasis;
  lmp: Date | null;
  scanDate: Date | null;
  scanWeeks: number | null;
  scanDays: number;
  dueDate: Date | null;
  weeks: number | null;
  days: number;
}

/** Gestational age in days on `today`, or null when the basis is incomplete / impossible. */
export function gestationalDays(a: PregnancyAnswers, today: Date): number | null {
  let ga: number | null = null;
  switch (a.basis) {
    case 'lmp':
      ga = a.lmp ? diffInDays(today, a.lmp) : null;
      break;
    case 'ultrasound':
      ga = a.scanDate && a.scanWeeks != null ? a.scanWeeks * 7 + a.scanDays + diffInDays(today, a.scanDate) : null;
      break;
    case 'due':
      ga = a.dueDate ? TERM_DAYS - diffInDays(a.dueDate, today) : null;
      break;
    case 'week':
      ga = a.weeks != null ? a.weeks * 7 + a.days : null;
      break;
  }
  if (ga == null || ga < 0 || ga > MAX_GA_DAYS) return null;
  return ga;
}

/** Weeks + days and the estimated due date for the summary card. */
export function pregnancySummary(a: PregnancyAnswers, today: Date): { weeks: number; days: number; due: Date } | null {
  const ga = gestationalDays(a, today);
  if (ga == null) return null;
  return { weeks: Math.floor(ga / 7), days: ga % 7, due: addDays(today, TERM_DAYS - ga) };
}

/**
 * The `POST /pregnancy/onboarding` body. The API knows LMP, ultrasound and a
 * manual week; a due date is sent as the LMP it implies (due − 280 days, the
 * same Naegele rule the backend dates with).
 */
export function pregnancyOnboardingBody(a: PregnancyAnswers, today: Date): OnboardingInput | null {
  if (gestationalDays(a, today) == null) return null;
  switch (a.basis) {
    case 'lmp':
      return { age_source: 'lmp', lmp_date: toApiDate(a.lmp!) };
    case 'ultrasound':
      return {
        age_source: 'ultrasound',
        ultrasound_date: toApiDate(a.scanDate!),
        ultrasound_weeks: Math.max(1, a.scanWeeks!),
        ultrasound_days: a.scanDays,
      };
    case 'due':
      return { age_source: 'lmp', lmp_date: toApiDate(addDays(a.dueDate!, -TERM_DAYS)) };
    case 'week':
      return { age_source: 'manual', manual_weeks: Math.max(1, a.weeks!), manual_days: a.days };
  }
}
