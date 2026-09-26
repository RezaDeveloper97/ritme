import { type CareItem, VISIT_STAGES, type VisitStage } from '@/entities/pregnancy';

/** `YYYY-MM` of a (year, month) in the locale's calendar — the API's `month`. */
export function monthKey(year: number, month: number): string {
  return `${String(year).padStart(4, '0')}-${String(month).padStart(2, '0')}`;
}

/** The stage a tap on the stepper moves to; `null` once the result is in. */
export function nextStage(stage: VisitStage | null): VisitStage | null {
  const i = stage == null ? 0 : VISIT_STAGES.indexOf(stage);
  return VISIT_STAGES[i + 1] ?? null;
}

/** How many of the three stepper segments are filled. */
export function stageProgress(stage: VisitStage | null): number {
  return stage == null ? 1 : VISIT_STAGES.indexOf(stage) + 1;
}

/** M3 AddAppointment prefilled from a care-plan row. */
export function bookHref(item: CareItem): string {
  const q = new URLSearchParams({ kind: 'in_person', title: item.title, care_item_key: item.key });
  const date = item.suggestedDate ?? item.date;
  if (date) q.set('date', date);
  return `/reminders/appointment/new?${q.toString()}`;
}

/** Maps search for a visit's place (opened in a new tab). */
export function directionsHref(place: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(place)}`;
}

/** Days covered by the doctor report: the last four weeks, today included. */
export const REPORT_DAYS = 28;
