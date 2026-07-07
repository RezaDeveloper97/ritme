import { TOTAL_WEEKS } from './config';

/**
 * Pure derivations for the pregnancy timeline. The backend already computes the
 * current week and trimester; these helpers turn that into display values
 * (progress bar, weeks-to-go, trimester grouping) with no I/O — so they are
 * unit-tested (`progress.test.ts`) per the frontend definition of done.
 */

export type Trimester = 1 | 2 | 3;

export interface PregnancyProgress {
  /** Current week clamped to the 1..40 display range. */
  currentWeek: number;
  trimester: Trimester;
  /** Whole weeks left until week 40 (never negative). */
  weeksRemaining: number;
  /** 0..100 progress across the 40-week timeline. */
  progressPct: number;
}

/** Standard obstetric trimester boundaries: T1 ≤12w, T2 13–27w, T3 ≥28w. */
export function trimesterOfWeek(week: number): Trimester {
  if (week <= 12) return 1;
  if (week <= 27) return 2;
  return 3;
}

/** Clamp any incoming week into the valid 1..40 display range. */
export function clampWeek(week: number): number {
  if (!Number.isFinite(week)) return 1;
  return Math.min(TOTAL_WEEKS, Math.max(1, Math.trunc(week)));
}

/**
 * Turn the backend's `current_week` into timeline display values. Returns `null`
 * when there is no active pregnancy week yet (pre-onboarding).
 */
export function derivePregnancyProgress(currentWeek: number | null): PregnancyProgress | null {
  if (currentWeek == null || !Number.isFinite(currentWeek) || currentWeek < 1) return null;

  const week = clampWeek(currentWeek);
  return {
    currentWeek: week,
    trimester: trimesterOfWeek(week),
    weeksRemaining: Math.max(0, TOTAL_WEEKS - week),
    progressPct: Math.round((week / TOTAL_WEEKS) * 100),
  };
}
