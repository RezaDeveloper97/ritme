import {
  type Appointment,
  type AppointmentKind,
  type Medication,
  type MedicationForm,
  type MedicationSchedule,
  medicationSchedule,
  type TodayDose,
} from '@/entities/care-reminder';
import type { IconName } from '@/shared/ui';

/*
 * Pure view-state for the Reminders screen (v13_Reminders) — no React, no
 * locale, so what each tab, card and row shows is unit-tested on its own.
 */

// ── Tabs ─────────────────────────────────────────────────────────

export const REMINDER_TABS = ['all', 'medications', 'appointments'] as const;
export type RemindersTab = (typeof REMINDER_TABS)[number];

/** `?tab=` → a tab; anything unknown (or absent) is «همه». */
export function parseTab(value: string | null | undefined): RemindersTab {
  return REMINDER_TABS.includes(value as RemindersTab) ? (value as RemindersTab) : 'all';
}

/** The URL for a tab — «همه» is the bare route so the default stays clean. */
export function tabHref(tab: RemindersTab): string {
  return tab === 'all' ? '/reminders' : `/reminders?tab=${tab}`;
}

/** Which sections a tab shows. The dose strip is medication data, so it goes with them. */
export function visibleSections(tab: RemindersTab): {
  today: boolean;
  medications: boolean;
  appointments: boolean;
} {
  return {
    today: tab !== 'appointments',
    medications: tab !== 'appointments',
    appointments: tab !== 'medications',
  };
}

/** Arrow-key movement across the tablist (wraps; RTL flips the arrows). */
export function nextTab(
  current: RemindersTab,
  key: string,
  dir: 'rtl' | 'ltr',
): RemindersTab | null {
  const i = REMINDER_TABS.indexOf(current);
  const last = REMINDER_TABS.length - 1;
  const forward = dir === 'rtl' ? 'ArrowLeft' : 'ArrowRight';
  const backward = dir === 'rtl' ? 'ArrowRight' : 'ArrowLeft';
  if (key === forward) return REMINDER_TABS[i === last ? 0 : i + 1];
  if (key === backward) return REMINDER_TABS[i === 0 ? last : i - 1];
  if (key === 'Home') return REMINDER_TABS[0];
  if (key === 'End') return REMINDER_TABS[last];
  return null;
}

// ── Slots ────────────────────────────────────────────────────────

/** Which part of the day a `HH:MM` slot falls in (`care.slotPeriod.*`). */
export type SlotPeriod = 'morning' | 'noon' | 'evening' | 'night';

function parseSlot(slot: string): { hour: number; minute: number } {
  const [h, m] = slot.split(':');
  const hour = Number(h);
  const minute = Number(m);
  return {
    hour: Number.isFinite(hour) ? Math.min(23, Math.max(0, hour)) : 0,
    minute: Number.isFinite(minute) ? Math.min(59, Math.max(0, minute)) : 0,
  };
}

/** 05–11 morning, 12–14 noon, 15–18 evening, 19–04 night (same as the home card). */
export function slotPeriod(slot: string): SlotPeriod {
  const { hour } = parseSlot(slot);
  if (hour >= 5 && hour < 12) return 'morning';
  if (hour >= 12 && hour < 15) return 'noon';
  if (hour >= 15 && hour < 19) return 'evening';
  return 'night';
}

/** `HH:MM` → 12-hour `h:MM`, Latin digits; the period word carries am/pm. */
export function slotClock(slot: string): string {
  const { hour, minute } = parseSlot(slot);
  const h12 = hour % 12 === 0 ? 12 : hour % 12;
  return `${h12}:${String(minute).padStart(2, '0')}`;
}

// ── «امروز» dose strip ───────────────────────────────────────────

export interface DoseCardState {
  /** Stable React key — one medication can have several slots a day. */
  key: string;
  reminderId: number;
  slot: string;
  title: string;
  taken: boolean;
  clock: string;
  period: SlotPeriod;
  /** What tapping the check does next. */
  nextTaken: boolean;
}

export function doseCardState(dose: TodayDose): DoseCardState {
  return {
    key: `${dose.reminderId}-${dose.slot}`,
    reminderId: dose.reminderId,
    slot: dose.slot,
    title: dose.title,
    taken: dose.taken,
    clock: slotClock(dose.slot),
    period: slotPeriod(dose.slot),
    nextTaken: !dose.taken,
  };
}

// ── Medication list ──────────────────────────────────────────────

/** Icon-tile tint per form (README "Colors"): capsules teal, the rest brand. */
export type MedicationTone = 'brand' | 'teal';

const FORM_ICON: Record<MedicationForm, IconName> = {
  tablet: 'tablet',
  capsule: 'capsule',
  syrup: 'glass',
  injection: 'pill',
  drops: 'drop',
};

export interface MedicationRowState {
  id: number;
  title: string;
  /** Server-derived «۴۰۰ میکروگرم», shown after the name when present. */
  dose: string | null;
  icon: IconName;
  tone: MedicationTone;
  form: MedicationForm;
  schedule: MedicationSchedule;
  /** Every slot of the day, in order. */
  slots: Array<{ clock: string; period: SlotPeriod }>;
  amount: number;
  isActive: boolean;
}

export function medicationRowState(med: Medication): MedicationRowState {
  return {
    id: med.id,
    title: med.title,
    dose: med.subtitle && med.subtitle.trim() !== '' ? med.subtitle : null,
    icon: FORM_ICON[med.form] ?? 'pill',
    tone: med.form === 'capsule' ? 'teal' : 'brand',
    form: med.form,
    schedule: medicationSchedule(med.weekdays),
    slots: med.times.map((slot) => ({ clock: slotClock(slot), period: slotPeriod(slot) })),
    amount: med.amount,
    isActive: med.isActive,
  };
}

/** Minutes since midnight of a medication's first slot (for ordering the list). */
function firstSlotMinutes(med: Medication): number {
  const first = [...med.times].sort()[0];
  if (!first) return 24 * 60;
  const { hour, minute } = parseSlot(first);
  return hour * 60 + minute;
}

/** List order (artboard): active reminders first, each group by its first dose of the day. */
export function sortMedications(meds: readonly Medication[]): Medication[] {
  return [...meds].sort(
    (a, b) =>
      Number(b.isActive) - Number(a.isActive) ||
      firstSlotMinutes(a) - firstSlotMinutes(b) ||
      a.id - b.id,
  );
}

// ── Appointment list ─────────────────────────────────────────────

/** Date-tile tint: amber for a visit, teal for a phone/online consultation. */
export type AppointmentTone = 'amber' | 'teal';

export function appointmentTone(kind: AppointmentKind): AppointmentTone {
  return kind === 'in_person' ? 'amber' : 'teal';
}

export interface AppointmentRowState {
  id: number;
  title: string;
  kind: AppointmentKind;
  tone: AppointmentTone;
  /** `Y-m-d` of the visit (null when the server sent none). */
  date: string | null;
  /** `HH:MM`, or null. */
  time: string | null;
  /** Who, appended to the title («{title} · {with}») — in-person visits only. */
  titleWith: string | null;
  /**
   * The third meta part. In person: the place (who is already in the title).
   * Phone / online: who, since the "place" is a number or a link.
   */
  detail: string | null;
}

export function appointmentRowState(appt: Appointment): AppointmentRowState {
  const [date, clock] = (appt.scheduledAt ?? '').split(' ');
  const who = appt.withWhom?.trim() || null;
  const place = appt.location?.trim() || null;
  const inPerson = appt.kind === 'in_person';
  return {
    id: appt.id,
    title: appt.title,
    titleWith: inPerson ? who : null,
    detail: inPerson ? place : (who ?? place),
    kind: appt.kind,
    tone: appointmentTone(appt.kind),
    date: date ? date : null,
    time: clock ? clock.slice(0, 5) : null,
  };
}

// ── Section status ───────────────────────────────────────────────

export type SectionStatus = 'loading' | 'error' | 'empty' | 'ready';

export function sectionStatus(query: {
  isLoading: boolean;
  isError: boolean;
  data: readonly unknown[] | undefined;
}): SectionStatus {
  if (query.isError) return 'error';
  if (query.isLoading || !query.data) return 'loading';
  return query.data.length === 0 ? 'empty' : 'ready';
}
