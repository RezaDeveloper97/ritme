import type { IvfCycle, IvfDose, IvfStage, IvfStageInfo, IvfTimelineStep } from '@/entities/ivf';

/*
 * Pure helpers of the IVF home (nbl_IVF_Home, CB-IVF-02). No React, no locale:
 * the UI turns the keys into copy.
 */

/**
 * Which IVF sub-screens exist yet. CB-IVF-03 (`/ivf/meds`), -04 (`/ivf/scan`)
 * and -05 (`/ivf/tww`) flip their flag; until then the home hides the link
 * (the «برنامه» link scrolls to today's injections instead) — never a 404.
 */
export const IVF_SCREENS_READY = { meds: true, scan: true, tww: true } as const;

/** Anchor of «تزریق‌های امروز» — the bottom-nav «درمان» placeholder target (IVF_TREATMENT_FALLBACK). */
export const DOSES_ANCHOR = 'ivf-doses';

/** Units the copy knows («۱۵۰ واحد»); anything else is shown as typed. */
const KNOWN_UNITS = ['iu', 'mg', 'mcg', 'ml'] as const;
export type DoseUnitKey = (typeof KNOWN_UNITS)[number] | 'other';

export function unitKey(unit: string | null): DoseUnitKey {
  const u = unit?.trim().toLowerCase() ?? '';
  return (KNOWN_UNITS as readonly string[]).includes(u) ? (u as DoseUnitKey) : 'other';
}

/** `true` when the dose row has an amount to show («۱۵۰ واحد»). */
export function hasAmount(dose: Pick<IvfDose, 'dose'>): boolean {
  return Boolean(dose.dose?.trim());
}

/** Catalog title/hint of a stage (admin-editable), else null → the bundled fallback copy. */
export function stageCopy(
  stage: IvfStage,
  catalog: readonly IvfStageInfo[] | undefined,
): { title: string | null; hint: string | null } {
  const row = catalog?.find((item) => item.code === stage);
  return { title: row?.title ?? null, hint: row?.body ?? null };
}

/**
 * What the step's second line shows: the stage hint on done/current steps (the
 * board), a known date on an upcoming step (e.g. the beta day), else nothing.
 */
export function stepMeta(step: IvfTimelineStep): 'hint' | 'date' | null {
  if (step.status !== 'todo') return 'hint';
  return step.date ? 'date' : null;
}

/** The «روز ۷ تحریک» chip: only for an open cycle with a known stage day. */
export function stageDayChip(cycle: IvfCycle): { day: number; stage: IvfStage } | null {
  if (cycle.status !== 'open' || cycle.stageDay === null || cycle.stageDay < 1) return null;
  return { day: cycle.stageDay, stage: cycle.stage };
}

/** `HH:MM` of a Tehran wall clock `Y-m-d H:i[:s]` (or of a dose slot). */
export function clockOf(value: string): string {
  const match = /(\d{2}):(\d{2})/.exec(value.length > 10 ? value.slice(10) : value);
  return match ? `${match[1]}:${match[2]}` : '';
}

/** `Y-m-d` of a wall clock. */
export function dayOf(value: string): string {
  return value.slice(0, 10);
}

/** «امروز / فردا / <date>» for the next appointment. */
export function whenKey(daysUntil: number | null): 'today' | 'tomorrow' | 'date' {
  if (daysUntil === 0) return 'today';
  if (daysUntil === 1) return 'tomorrow';
  return 'date';
}
