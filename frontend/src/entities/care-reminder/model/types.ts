/**
 * Care reminders (M3): medications, doctor appointments and today's doses.
 * Contract: docs/care-reminders/README.md (`/api/v1/care/*`, Go only).
 *
 * Everything here is personal health data (§11) — display only, never logged,
 * never put in analytics, error reports or URLs beyond what the API needs.
 *
 * Dates cross the boundary as the API sends them (`Y-m-d`, Tehran wall-clock
 * `Y-m-d H:i:s`) and are converted to the locale calendar only for display
 * (`shared/lib/date`, §7).
 */

export const MEDICATION_FORMS = ['tablet', 'capsule', 'syrup', 'injection', 'drops'] as const;
export type MedicationForm = (typeof MEDICATION_FORMS)[number];

/** Suggested units; the API also accepts free text, so `unit` stays a string. */
export const MEDICATION_UNITS = ['mg', 'mcg', 'ml', 'iu', 'drop'] as const;
export type MedicationUnit = (typeof MEDICATION_UNITS)[number];

export const MEDICATION_DURATIONS = ['ongoing', 'until_date', 'pregnancy_end'] as const;
export type MedicationDuration = (typeof MEDICATION_DURATIONS)[number];

export const APPOINTMENT_KINDS = ['in_person', 'phone', 'online'] as const;
export type AppointmentKind = (typeof APPOINTMENT_KINDS)[number];

export const APPOINTMENT_TOPICS = ['ultrasound', 'checkup', 'lab', 'consult', 'vaccine', 'other'] as const;
export type AppointmentTopic = (typeof APPOINTMENT_TOPICS)[number];

export const REMIND_BEFORE = ['1h', '3h', '1d', '2d'] as const;
export type RemindBefore = (typeof REMIND_BEFORE)[number];

export const APPOINTMENT_STATUSES = ['scheduled', 'cancelled'] as const;
export type AppointmentStatus = (typeof APPOINTMENT_STATUSES)[number];

/** Saturday-based weekday index (Saturday = 0 … Friday = 6), the project week start. */
export type Weekday = 0 | 1 | 2 | 3 | 4 | 5 | 6;
export const ALL_WEEKDAYS: readonly Weekday[] = [0, 1, 2, 3, 4, 5, 6];

/** A `reminders` row of type `medication`, with its `meta` flattened in. */
export interface Medication {
  id: number;
  /** Medication name («فولیک اسید»). */
  title: string;
  /** Server-derived "dose unit" («۴۰۰ میکروگرم»). */
  subtitle: string | null;
  dose: string;
  unit: string;
  form: MedicationForm;
  /** 1–4 sorted `HH:MM` slots. */
  times: string[];
  weekdays: Weekday[];
  /** Units per dose (e.g. 1 tablet). */
  amount: number;
  duration: MedicationDuration;
  notify: boolean;
  startsOn: string | null;
  endsOn: string | null;
  isActive: boolean;
  notes: string | null;
}

export interface PrepItem {
  id: string;
  text: string;
  done: boolean;
}

/** A `reminders` row of type `appointment`, with its `meta` flattened in. */
export interface Appointment {
  id: number;
  title: string;
  /** "who · specialty". */
  subtitle: string | null;
  kind: AppointmentKind;
  /** Doctor / counsellor name (`meta.with`). */
  withWhom: string | null;
  specialty: string | null;
  topic: AppointmentTopic;
  location: string | null;
  remindBefore: RemindBefore;
  addToCalendar: boolean;
  prep: PrepItem[];
  status: AppointmentStatus;
  /** Tehran wall-clock `Y-m-d H:i:s`. */
  scheduledAt: string | null;
  /** The reminder (notification) switch. */
  isActive: boolean;
  notes: string | null;
  /** Pregnancy care-plan item this visit books (M7), e.g. `nt_scan`. */
  careItemKey?: string | null;
  /** Pregnancy visit stage (`booked` | `done` | `result`), null outside pregnancy. */
  stage?: string | null;
  resultNote?: string | null;
}

export type AppointmentScope = 'upcoming' | 'past' | 'all';

/** One slot of one active medication on the requested day. */
export interface TodayDose {
  reminderId: number;
  title: string;
  form: MedicationForm;
  slot: string;
  taken: boolean;
}

export interface NextAppointment {
  id: number;
  kind: AppointmentKind;
  title: string;
  withWhom: string | null;
  scheduledAt: string;
  daysUntil: number;
  location: string | null;
  remindBefore: RemindBefore;
  /** The appointment's reminder bell: off → no «… قبل یادآوری» on the home card. */
  isActive: boolean;
}

/** GET /care/today. */
export interface CareToday {
  date: string;
  doses: TodayDose[];
  takenCount: number;
  total: number;
  nextAppointment: NextAppointment | null;
}

export interface CareOption<T extends string = string> {
  value: T;
  label: string;
}

/** GET /care/enums — labels localized by Accept-Language. */
export interface CareEnums {
  forms: CareOption<MedicationForm>[];
  units: CareOption[];
  durations: CareOption<MedicationDuration>[];
  kinds: CareOption<AppointmentKind>[];
  topics: CareOption<AppointmentTopic>[];
  remindBefore: CareOption<RemindBefore>[];
}
