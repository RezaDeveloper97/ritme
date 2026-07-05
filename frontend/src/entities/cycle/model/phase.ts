import { diffInDays } from '@/shared/lib/date';

import type { CycleConfig, CycleDayInfo, CyclePhase, PregnancyChance } from './types';

// The luteal phase (ovulation → next period) is biologically stable at ~14 days,
// so ovulation is estimated as `cycleLength - 14`. The fertile window is the
// five days before ovulation plus the day itself. These are estimates for a
// tracking aid, not a medical determination (CLAUDE.md §11).
const LUTEAL_LENGTH = 14;
const FERTILE_DAYS_BEFORE_OVULATION = 5;

/** Positive modulo so days in past cycles still map into 0 … length-1. */
function wrap(value: number, length: number): number {
  return ((value % length) + length) % length;
}

/**
 * Classify a single date against a cycle. Pure and locale-independent: it takes
 * plain `Date`s and returns the phase, the 1-based day within the cycle, and a
 * relative conception likelihood. Repeats every `cycleLength` days in both
 * directions, so past and predicted cycles are handled the same way.
 */
export function cycleDayInfo(date: Date, config: CycleConfig): CycleDayInfo {
  const { startDate, cycleLength, periodLength } = config;

  const position = wrap(diffInDays(date, startDate), cycleLength); // 0-based
  const cycleDay = position + 1;
  const ovulation = cycleLength - LUTEAL_LENGTH; // 0-based index in the cycle
  const fertileStart = ovulation - FERTILE_DAYS_BEFORE_OVULATION;

  let phase: CyclePhase;
  let pregnancyChance: PregnancyChance;

  if (position < periodLength) {
    phase = 'period';
    pregnancyChance = 'low';
  } else if (position === ovulation) {
    phase = 'ovulation';
    pregnancyChance = 'high';
  } else if (position >= fertileStart && position <= ovulation + 1) {
    phase = 'fertile';
    pregnancyChance = 'medium';
  } else if (position < ovulation) {
    phase = 'follicular';
    pregnancyChance = 'low';
  } else {
    phase = 'luteal';
    pregnancyChance = 'low';
  }

  return { cycleDay, phase, pregnancyChance };
}
