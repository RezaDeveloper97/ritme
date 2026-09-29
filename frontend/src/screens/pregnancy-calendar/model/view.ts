import type { AppointmentTopic } from '@/entities/care-reminder';
import { type CareItem, type CareItemKind, VISIT_STAGES, type VisitStage } from '@/entities/pregnancy';

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

/** The AddAppointment topic («در خصوص چه چیزی؟») a care-plan kind books. */
const TOPIC_BY_CARE_KIND: Record<CareItemKind, AppointmentTopic> = {
  scan: 'ultrasound',
  test: 'lab',
  vaccine: 'vaccine',
  visit: 'checkup',
};

/** The topic to prefill for a care item; `checkup` (the form default) when the kind is unknown. */
export function careItemTopic(kind: CareItemKind | null | undefined): AppointmentTopic {
  return (kind && TOPIC_BY_CARE_KIND[kind]) || 'checkup';
}

/**
 * M3 AddAppointment for a care-plan row. The URL carries only the visit kind
 * and the allow-listed way back (T-M7-20 `return_to`); what the visit is for
 * — the item, its topic and date, i.e. that the user is pregnant and which
 * test is due — travels as a one-time prefill (`shared/lib/handoff`), never in
 * the query string (§11, security audit M3-M7 #3).
 */
export const BOOK_HREF = '/reminders/appointment/new?kind=in_person&return_to=/pregnancy/calendar';

/**
 * The prefill for a care-plan row. `topic` is what the visit is for (T-M7-17),
 * so the NT scan opens as «سونوگرافی», not «ویزیت دوره‌ای».
 */
export function bookPrefill(item: CareItem): Record<string, string> {
  const prefill: Record<string, string> = {
    topic: careItemTopic(item.kind),
    title: item.title,
    careItemKey: item.key,
  };
  const date = item.suggestedDate ?? item.date;
  if (date) prefill.date = date;
  return prefill;
}

/** Maps search for a visit's place (opened in a new tab). */
export function directionsHref(place: string): string {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(place)}`;
}

/** Days covered by the doctor report: the last four weeks, today included. */
export const REPORT_DAYS = 28;
