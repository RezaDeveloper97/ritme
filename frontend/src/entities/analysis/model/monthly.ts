import type { AnalysisPhrase, SymptomCount } from './types';

/**
 * `GET /analysis/monthly/:ym[?calendar=]` (B-N3-07 engine, B-N3-10 screen) after
 * the boundary parser in `api/monthly.ts`. Personal health data (CLAUDE.md §11):
 * never log these.
 */

/** Calendar of the `ym` path segment. */
export const MONTHLY_CALENDARS = ['gregorian', 'jalali'] as const;
export type MonthlyCalendar = (typeof MONTHLY_CALENDARS)[number];

/** Rows of «شاخص / این ماه / تغییر», in the order the server sends them. */
export const MONTHLY_METRIC_KEYS = ['cycle_length', 'days_logged', 'sleep', 'weight', 'blood_pressure', 'good_mood'] as const;
export type MonthlyMetricKey = (typeof MONTHLY_METRIC_KEYS)[number];

export interface BloodPressureValue {
  systolic: number;
  diastolic: number;
}

/** One table row; `value`/`delta` are null without data (or without last month's data for the delta). */
export interface MonthlyMetric {
  key: MonthlyMetricKey;
  value: number | BloodPressureValue | null;
  delta: number | BloodPressureValue | null;
  unit: string | null;
}

export interface MonthlyMonth {
  /** `YYYY-MM` in {@link MonthlyMonth.calendar}. */
  key: string;
  calendar: MonthlyCalendar;
  /** Gregorian `YYYY-MM-DD`; `to` is clipped to today for the running month. */
  from: string;
  to: string;
  complete: boolean;
}

export interface MonthlyReport {
  month: MonthlyMonth;
  headline: AnalysisPhrase;
  summary: { parts: AnalysisPhrase[]; text: string };
  metrics: MonthlyMetric[];
  /** Up to 3, most frequent first. */
  topSymptoms: SymptomCount[];
  suggestion: AnalysisPhrase;
  /** «ساخت PDF برای پزشک» — a Plus feature (`plus.pdf_share`). */
  pdf: { plus: boolean; locked: boolean };
}

export function isMonthlyCalendar(value: unknown): value is MonthlyCalendar {
  return typeof value === 'string' && (MONTHLY_CALENDARS as readonly string[]).includes(value);
}
