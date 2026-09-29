import {
  APPOINTMENT_KINDS,
  APPOINTMENT_TOPICS,
  type Appointment,
  type AppointmentKind,
  type AppointmentTopic,
  type PrepItem,
  type RemindBefore,
} from '@/entities/care-reminder';

/*
 * Pure form state for v13_AddAppointment — parsing `?kind=`, seeding from an
 * existing appointment, and the prep textarea ⇄ checklist mapping.
 */

export interface AppointmentFormState {
  kind: AppointmentKind;
  withWhom: string;
  specialty: string;
  topic: AppointmentTopic;
  title: string;
  /** `Y-m-d` (Gregorian, API) or '' until picked. */
  date: string;
  /** `HH:MM` or '' until picked. */
  time: string;
  location: string;
  remindBefore: RemindBefore;
  addToCalendar: boolean;
  /** One line = one checklist item. */
  prepText: string;
  /** Pregnancy care-plan item this visit books (from the prefill), '' when none. */
  careItemKey: string;
}

/**
 * What a new appointment may be prefilled with. Only `kind` comes from the
 * query string; the rest arrives as a one-time handoff (`?prefill=<id>`,
 * `shared/lib/handoff`) so no title, topic or date ever sits in a URL (§11,
 * security audit M3-M7 #3).
 */
export interface AppointmentPrefill {
  kind?: string | null;
  /** What the visit is for (a care-plan «رزرو» sends scan → `ultrasound`, …). */
  topic?: string | null;
  title?: string | null;
  date?: string | null;
  careItemKey?: string | null;
}

/** Handoff fields → prefill. Unknown keys are ignored; values are still validated by `formFromPrefill`. */
export function prefillFromHandoff(data: Readonly<Record<string, string>> | null): AppointmentPrefill {
  if (!data) return {};
  const { topic, title, date, careItemKey } = data;
  return { topic, title, date, careItemKey };
}

/** Local `YYYY-MM-DD` of a Date. */
export function isoDay(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

/** A prefill date → a valid `YYYY-MM-DD` that is not before `today`, else ''. */
export function parsePrefillDate(value: string | null | undefined, today: string): string {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return '';
  const d = new Date(`${value}T00:00:00`);
  if (Number.isNaN(d.getTime()) || isoDay(d) !== value) return '';
  return value < today ? '' : value;
}

/** A fresh form seeded from `?kind=` plus the handoff's topic / title / date / care-plan key. */
export function formFromPrefill(prefill: AppointmentPrefill, today: string): AppointmentFormState {
  const key = (prefill.careItemKey ?? '').trim();
  return {
    ...emptyForm(parseKind(prefill.kind)),
    topic: parseTopic(prefill.topic),
    title: (prefill.title ?? '').trim().slice(0, 120),
    date: parsePrefillDate(prefill.date, today),
    careItemKey: /^[a-z0-9_.-]{1,64}$/i.test(key) ? key : '',
  };
}

export function parseKind(value: string | null | undefined): AppointmentKind {
  return APPOINTMENT_KINDS.includes(value as AppointmentKind) ? (value as AppointmentKind) : 'in_person';
}

/** A prefill topic → a known topic, else the form's default (`checkup`). */
export function parseTopic(value: string | null | undefined): AppointmentTopic {
  return APPOINTMENT_TOPICS.includes(value as AppointmentTopic) ? (value as AppointmentTopic) : 'checkup';
}

export function emptyForm(kind: AppointmentKind): AppointmentFormState {
  return {
    kind,
    withWhom: '',
    specialty: '',
    topic: 'checkup',
    title: '',
    date: '',
    time: '',
    location: '',
    remindBefore: '1d',
    addToCalendar: true,
    prepText: '',
    careItemKey: '',
  };
}

export function formFromAppointment(appt: Appointment): AppointmentFormState {
  const [date = '', clock = ''] = (appt.scheduledAt ?? '').split(' ');
  return {
    kind: appt.kind,
    withWhom: appt.withWhom ?? '',
    specialty: appt.specialty ?? '',
    topic: appt.topic,
    title: appt.title ?? '',
    date,
    time: clock.slice(0, 5),
    location: appt.location ?? '',
    remindBefore: appt.remindBefore,
    addToCalendar: appt.addToCalendar,
    prepText: appt.prep.map((item) => item.text).join('\n'),
    careItemKey: appt.careItemKey ?? '',
  };
}

/**
 * Textarea → checklist. A line that matches an existing item keeps its id and
 * tick, so editing the list doesn't reset what was already prepared.
 */
export function prepFromText(text: string, existing: PrepItem[] = [], makeId = defaultId): PrepItem[] {
  const pool = [...existing];
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const i = pool.findIndex((item) => item.text.trim() === line);
      if (i >= 0) {
        const [kept] = pool.splice(i, 1);
        return { ...kept!, text: line };
      }
      return { id: makeId(), text: line, done: false };
    });
}

function defaultId(): string {
  return `p${Date.now().toString(36)}${Math.random().toString(36).slice(2, 7)}`;
}

export type FormError = 'with' | 'date' | 'time';

/** The first missing required field, or null when the form can be saved. */
export function validateForm(state: AppointmentFormState): FormError | null {
  if (!state.withWhom.trim()) return 'with';
  if (!state.date) return 'date';
  if (!/^\d{2}:\d{2}$/.test(state.time)) return 'time';
  return null;
}

/**
 * In-app screens a saved appointment may return to (`?return_to=`). An
 * allow-list of exact paths — never a free-form redirect (open redirect).
 */
export const RETURN_PATHS = ['/pregnancy/calendar'] as const;
export type ReturnPath = (typeof RETURN_PATHS)[number];

/** `?return_to=` → an allowed in-app path, else null. */
export function parseReturnTo(value: string | null | undefined): ReturnPath | null {
  return RETURN_PATHS.find((path) => path === value) ?? null;
}

/**
 * Where to go after saving (and the header's back link): an allowed
 * `?return_to=`, else — for a new visit booked from the pregnancy care plan
 * (a prefilled care-plan key, only the pregnancy calendar sends it) — the calendar;
 * null → the default (the appointment's detail page).
 */
export function returnPathFor(returnTo: string | null | undefined, isNew: boolean, careItemKey: string): ReturnPath | null {
  return parseReturnTo(returnTo) ?? (isNew && careItemKey ? '/pregnancy/calendar' : null);
}
