import {
  APPOINTMENT_KINDS,
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
}

export function parseKind(value: string | null | undefined): AppointmentKind {
  return APPOINTMENT_KINDS.includes(value as AppointmentKind) ? (value as AppointmentKind) : 'in_person';
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
    addToCalendar: false,
    prepText: '',
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
