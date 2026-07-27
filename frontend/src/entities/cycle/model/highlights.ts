import { normalizePhase } from './predictions';
import type { CycleCalculation } from './types';

/**
 * A notable state a single cycle day can be in — surfaced as its own badge on
 * the home info card so a tapped day shows not just its phase but every relevant
 * signal (fertile window, PMS, imminent period). Stable codes; the UI maps each
 * to a localized label + icon. Informational only, not medical advice (§11).
 */
export type CycleDayHighlight = 'period' | 'fertile' | 'ovulation' | 'pms' | 'period_tomorrow';

/**
 * Derive the ordered highlights for a day's calculation. Ovulation sits inside
 * the fertile window, so it wins over the plain `fertile` badge; PMS and an
 * imminent period can co-occur with the luteal phase and are shown alongside.
 * Pure and locale-free (CLAUDE.md §7) so it stays unit-testable.
 */
export function deriveDayHighlights(calc: CycleCalculation): CycleDayHighlight[] {
  const phase = normalizePhase(calc.phase);
  const out: CycleDayHighlight[] = [];
  if (phase === 'period') out.push('period');
  if (phase === 'ovulation') out.push('ovulation');
  else if (calc.isFertileWindow) out.push('fertile');
  if (calc.isPmsWindow) out.push('pms');
  if (calc.isPeriodTomorrow) out.push('period_tomorrow');
  return out;
}
