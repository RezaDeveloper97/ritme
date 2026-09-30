import type { CycleView, MainPhase } from '@/entities/cycle';
import type { PhaseSectionKey } from '@/entities/phase-content';
import { diffInDays, fromApiDate } from '@/shared/lib/date';

/**
 * The phase sheet's five tabs (`nbl_Cycle_Phase`) and which admin-edited
 * phase-content sections each one shows, in order. The content itself stays
 * DB-driven (phase_contents); only the grouping lives here. A tab whose
 * sections are all empty for the phase is hidden.
 */
export const PHASE_TABS = ['body', 'mood', 'nutrition', 'movement', 'relationship'] as const;
export type PhaseTab = (typeof PHASE_TABS)[number];

export const TAB_SECTIONS: Record<PhaseTab, readonly PhaseSectionKey[]> = {
  body: ['hormonal_changes', 'symptom_prediction', 'vaginal_discharge'],
  mood: ['sleep', 'skin_care'],
  nutrition: ['nutrition'],
  movement: ['exercise'],
  relationship: ['sex_tips', 'fertility'],
};

/** Tabs that have at least one non-empty section, in canonical order. */
export function availableTabs(sections: Partial<Record<PhaseSectionKey, string>>): PhaseTab[] {
  return PHASE_TABS.filter((tab) => TAB_SECTIONS[tab].some((k) => (sections[k] ?? '').trim() !== ''));
}

/** Phases with a named day range; `period_expected` / `unknown` have none. */
const RANGED: readonly MainPhase[] = ['menstrual', 'follicular', 'fertile', 'luteal'];

/**
 * Cycle-day range of the current main phase, from the engine's anchors and
 * effective lengths (the same numbers the calendar paints): period = day 1 …
 * period length, fertile = the 5 days before the estimated ovulation through
 * ovulation (the §19 display window), follicular in between, luteal after
 * ovulation to the cycle's end. Null when the engine can't place it.
 */
export function phaseDayRange(view: CycleView | null | undefined): { from: number; to: number } | null {
  if (!view?.mainPhase || !RANGED.includes(view.mainPhase)) return null;
  const start = view.anchors?.currentPeriodStart;
  const ovulation = view.anchors?.estimatedOvulationDate;
  const cycleLength = view.metrics?.effectiveCycleLength;
  const periodLength = view.metrics?.effectivePeriodLength;
  if (!start || !cycleLength || !periodLength) return null;
  const ovDay = ovulation ? diffInDays(fromApiDate(ovulation), fromApiDate(start)) + 1 : cycleLength - 14;
  const fertileFrom = Math.max(periodLength + 1, ovDay - 5);
  const ranges: Record<string, [number, number]> = {
    menstrual: [1, periodLength],
    follicular: [periodLength + 1, fertileFrom - 1],
    fertile: [fertileFrom, ovDay],
    luteal: [ovDay + 1, cycleLength],
  };
  const [from, to] = ranges[view.mainPhase];
  return from >= 1 && to >= from ? { from, to } : null;
}
