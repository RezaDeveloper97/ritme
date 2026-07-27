import type { WellbeingTone } from './types';

/**
 * Buckets the week's overall wellbeing score into the tone its caption is
 * written for. Pure and locale-free (§7) so it can be unit-tested.
 *
 * `null` — nothing scoreable was logged — is deliberately its own tone: the card
 * then invites the user to log instead of judging a week it knows nothing about.
 */
export function weeklyWellbeingTone(overallPercent: number | null): WellbeingTone {
  if (overallPercent === null) return 'none';
  if (overallPercent >= 80) return 'great';
  if (overallPercent >= 65) return 'good';
  if (overallPercent >= 45) return 'mixed';
  return 'low';
}

/**
 * Direction of a metric's week-over-week change, ignoring swings small enough
 * to be noise (a single differently-worded mood entry moves the average a few
 * points on its own).
 */
export function wellbeingTrend(delta: number | null): 'up' | 'down' | null {
  if (delta === null || Math.abs(delta) < 5) return null;
  return delta > 0 ? 'up' : 'down';
}
