import type {
  AppointmentKind,
  AppointmentTopic,
  PrepItem,
  RemindBefore,
} from '@/entities/care-reminder';

/** What the appointment form (v13_AddAppointment) submits. */
export interface AppointmentInput {
  kind: AppointmentKind;
  withWhom: string;
  specialty?: string | null;
  topic: AppointmentTopic;
  /** توضیح کوتاه — the server falls back to the topic label when empty. */
  title?: string | null;
  /** `Y-m-d` (Gregorian, API format). */
  date: string;
  /** `HH:MM`. */
  time: string;
  location?: string | null;
  remindBefore: RemindBefore;
  addToCalendar: boolean;
  /** «چیزهایی که باید آماده کنم» — free text. */
  notes?: string | null;
  prep?: PrepItem[];
}

export type AppointmentPatch = Partial<AppointmentInput> & { isActive?: boolean };

const blankToNull = (v: string | null | undefined): string | null => v?.trim() || null;

/**
 * camelCase input → the API body; undefined fields are omitted so a PUT stays
 * partial. Date + time become `scheduled_at` (Tehran wall-clock
 * `Y-m-d H:i:s`) only when both are present.
 */
export function toAppointmentBody(input: AppointmentPatch): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.kind !== undefined) body.kind = input.kind;
  if (input.withWhom !== undefined) body.with = input.withWhom.trim();
  if (input.specialty !== undefined) body.specialty = blankToNull(input.specialty);
  if (input.topic !== undefined) body.topic = input.topic;
  if (input.title !== undefined) body.title = blankToNull(input.title);
  if (input.date !== undefined && input.time !== undefined) {
    body.scheduled_at = `${input.date} ${input.time.slice(0, 5)}:00`;
  }
  if (input.location !== undefined) body.location = blankToNull(input.location);
  if (input.remindBefore !== undefined) body.remind_before = input.remindBefore;
  if (input.addToCalendar !== undefined) body.add_to_calendar = input.addToCalendar;
  if (input.notes !== undefined) body.notes = blankToNull(input.notes);
  if (input.prep !== undefined) {
    body.prep = input.prep
      .filter((item) => item.text.trim() !== '')
      .map((item) => ({ id: item.id, text: item.text.trim(), done: item.done }));
  }
  if (input.isActive !== undefined) body.is_active = input.isActive;
  return body;
}

/** The prep list with one item ticked/unticked (immutable). */
export function setPrepItemDone(prep: PrepItem[], itemId: string, done: boolean): PrepItem[] {
  return prep.map((item) => (item.id === itemId ? { ...item, done } : item));
}
