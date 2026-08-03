/** What a reminder is about — drives the row icon and form options. */
export type ReminderType = 'doctor' | 'medication' | 'appointment' | 'custom';

/** How often a reminder repeats. */
export type ReminderRecurrence = 'none' | 'daily' | 'weekly' | 'monthly';

/**
 * A user reminder (GET /reminders). Dates cross the boundary as Gregorian
 * ISO strings and are converted to the locale's calendar for display (§7).
 * Content is personal health context — never log it (§11).
 */
export interface Reminder {
  id: string;
  type: ReminderType;
  title: string;
  subtitle: string | null;
  notes: string | null;
  /** One-off datetime (ISO), when the reminder isn't recurring. */
  scheduledAt: string | null;
  recurrence: ReminderRecurrence;
  /** Time of day for recurring reminders, normalized to "HH:MM". */
  recurrenceTime: string | null;
  startsOn: string | null;
  endsOn: string | null;
  isActive: boolean;
}

/** A backend-localized option from GET /reminders/enums. */
export interface ReminderRecurrenceOption {
  value: ReminderRecurrence;
  label: string;
}

/** Type option carries the backend's suggested icon name as plain string. */
export interface ReminderTypeOption {
  value: ReminderType;
  label: string;
  icon: string;
}

/** Form options for the create/edit UI (labels localized by Accept-Language). */
export interface ReminderEnums {
  types: ReminderTypeOption[];
  recurrences: ReminderRecurrenceOption[];
}
