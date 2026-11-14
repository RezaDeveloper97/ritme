/**
 * Which of the three Night & Bloom hero variants the cycle home shows
 * (nbl_Cycle_Home / _Near / _During, B-N1-06) and which prompt rides on it.
 * Pure: the numbers come from the engine's `cycle_view` (§35), the dismissals
 * from the per-day «هنوز نه» / «هنوز ادامه دارد» taps.
 */
export type HeroVariant = 'normal' | 'near' | 'late' | 'during';

export interface HeroInput {
  /** The ring shows today (prompts only ever act on today). */
  isToday: boolean;
  inPeriod: boolean;
  /** Days until the predicted period (`days_to_period`), null when unknown. */
  daysLeft: number | null;
  /** `days_late` (> 0 once the predicted start has passed without a period). */
  daysLate: number | null;
  /** The current period has no confirmed end yet (`current_period_end_is_confirmed` false). */
  periodOpen: boolean;
  cycleDay: number | null;
  /** The usual period length (effective period duration). */
  periodLength: number;
  /** «هنوز نه» was tapped today. */
  startDismissed: boolean;
  /** «هنوز ادامه دارد» was tapped today. */
  endDismissed: boolean;
}

export interface HeroState {
  variant: HeroVariant;
  /** «پریودم شروع شد / هنوز نه». */
  startPrompt: boolean;
  /** «پریودم تموم شد / هنوز ادامه دارد». */
  endPrompt: boolean;
  /** «پایان پریود هنوز ثبت نشده» — the open period ran past its usual length. */
  endMissing: boolean;
}

/** Days before the predicted start that count as «نزدیک پریود». */
export const NEAR_PERIOD_DAYS = 2;

/** «پریودم تموم شد» shows from this many days before the usual period length (engine: EPL − 1). */
export const END_PROMPT_LEAD_DAYS = 1;

export function heroState(i: HeroInput): HeroState {
  if (i.inPeriod) {
    // Same window as the engine's `ring_action: end` (§2.43): from the day before the usual length on.
    // An unknown cycle day keeps the prompt, so an open period can always be closed.
    const endDue = i.cycleDay == null || i.cycleDay >= i.periodLength - END_PROMPT_LEAD_DAYS;
    const endPrompt = i.isToday && i.periodOpen && endDue && !i.endDismissed;
    return {
      variant: 'during',
      startPrompt: false,
      endPrompt,
      endMissing: endPrompt && i.cycleDay != null && i.cycleDay > i.periodLength,
    };
  }
  let variant: HeroVariant = 'normal';
  if (i.daysLate != null && i.daysLate > 0) variant = 'late';
  else if (i.daysLeft != null && i.daysLeft >= 0 && i.daysLeft <= NEAR_PERIOD_DAYS) variant = 'near';
  return {
    variant,
    startPrompt: variant !== 'normal' && i.isToday && !i.startDismissed,
    endPrompt: false,
    endMissing: false,
  };
}

/**
 * The page's phase from the engine's §35 `main_phase` — the same value the
 * day-status card and `/fertility/*` read — so the hero and the phase card
 * can't disagree with it. `period_expected` (predicted start reached, no
 * bleeding logged) is still the luteal phase; the «نزدیک پریود» variant words
 * it. Null for `unknown` / no anchor: the caller falls back to the legacy calc.
 */
export function phaseFromMainPhase(
  main: string | null | undefined,
): 'period' | 'follicular' | 'fertile' | 'luteal' | null {
  switch (main) {
    case 'menstrual':
      return 'period';
    case 'follicular':
    case 'fertile':
    case 'luteal':
      return main;
    case 'period_expected':
      return 'luteal';
    default:
      return null;
  }
}
