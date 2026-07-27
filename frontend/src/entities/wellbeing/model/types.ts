/** The three self-reported metrics the weekly summary scores. */
export type WellbeingMetricKey = 'mood' | 'sleep' | 'energy';

/** One tile of «خلاصه هفته», already scored by the backend. */
export interface WellbeingMetric {
  key: WellbeingMetricKey;
  /** 0-100 average over the window, or `null` when nothing scoreable was logged. */
  percent: number | null;
  /** Same average over the previous window; `null` when that week was empty. */
  previousPercent: number | null;
  /** `percent - previousPercent`, or `null` when either side is missing. */
  delta: number | null;
}

/**
 * The last seven days of mood / sleep / energy, averaged server-side from the
 * user's daily health logs (the app never scores logs itself — §8.1).
 */
export interface WeeklyWellbeing {
  metrics: WellbeingMetric[];
  /** Inclusive window, Gregorian `YYYY-MM-DD` — formatted by the date layer (§7). */
  from: string;
  to: string;
  /** Days in the window with a saved log; 0 means "nothing to summarize yet". */
  loggedDays: number;
  previousLoggedDays: number;
  /** Average of the metrics that could be scored, or `null` when none could. */
  overallPercent: number | null;
}

/** How the week went overall — drives the caption under the tiles. */
export type WellbeingTone = 'none' | 'low' | 'mixed' | 'good' | 'great';
