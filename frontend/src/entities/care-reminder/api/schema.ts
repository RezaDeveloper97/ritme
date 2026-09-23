import { z } from 'zod';

import {
  ALL_WEEKDAYS,
  APPOINTMENT_KINDS,
  APPOINTMENT_STATUSES,
  APPOINTMENT_TOPICS,
  type Appointment,
  type CareEnums,
  type CareOption,
  type CareToday,
  MEDICATION_DURATIONS,
  MEDICATION_FORMS,
  type Medication,
  REMIND_BEFORE,
  type Weekday,
} from '../model/types';

/**
 * Boundary parsers for `/api/v1/care/*` (CLAUDE.md §10 — zod at the boundary).
 *
 * Tolerance rules, so a newer backend can never crash an older bundle:
 * - an enum value this bundle doesn't know falls back to a safe default
 *   (`tablet`, `in_person`, `other`, `1d`, `ongoing`, `scheduled`);
 * - a row may carry its `meta` nested (`{…, meta: {dose, …}}`) or flattened —
 *   both are read, top-level columns win;
 * - ids may arrive as numbers or numeric strings.
 */

const idSchema = z.union([z.number(), z.string()]).pipe(z.coerce.number().int());

const nullableText = z
  .string()
  .nullish()
  .transform((v) => (v ? v : null));

/** "08:00" or "08:00:00" → "08:00". */
const slotSchema = z.string().transform((v) => v.slice(0, 5));

const formSchema = z.enum(MEDICATION_FORMS).catch('tablet');
const durationSchema = z.enum(MEDICATION_DURATIONS).catch('ongoing');
const kindSchema = z.enum(APPOINTMENT_KINDS).catch('in_person');
const topicSchema = z.enum(APPOINTMENT_TOPICS).catch('other');
const remindBeforeSchema = z.enum(REMIND_BEFORE).catch('1d');
const statusSchema = z.enum(APPOINTMENT_STATUSES).catch('scheduled');

const weekdaysSchema = z
  .array(z.unknown())
  .catch([])
  .transform((list): Weekday[] => {
    const valid = new Set<Weekday>();
    for (const v of list) {
      const n = typeof v === 'string' ? Number(v) : v;
      if (typeof n === 'number' && (ALL_WEEKDAYS as readonly number[]).includes(n)) {
        valid.add(n as Weekday);
      }
    }
    return [...valid].sort((a, b) => a - b);
  });

/** Merge a nested `meta` object (object or JSON string) under the row's columns. */
function flattenMeta(raw: unknown): unknown {
  if (typeof raw !== 'object' || raw === null) return raw;
  const row = raw as Record<string, unknown>;
  let meta: unknown = row.meta;
  if (typeof meta === 'string') {
    try {
      meta = JSON.parse(meta) as unknown;
    } catch {
      meta = null;
    }
  }
  if (typeof meta !== 'object' || meta === null) return row;
  const merged: Record<string, unknown> = { ...(meta as Record<string, unknown>) };
  for (const [key, value] of Object.entries(row)) {
    if (key !== 'meta' && value !== undefined) merged[key] = value;
  }
  return merged;
}

const medicationRowSchema = z.object({
  id: idSchema,
  title: z.string(),
  subtitle: nullableText,
  dose: z.union([z.string(), z.number()]).nullish().transform((v) => (v == null ? '' : String(v))),
  unit: z.string().nullish().transform((v) => v ?? ''),
  form: formSchema.default('tablet'),
  times: z
    .array(slotSchema)
    .catch([])
    .default([])
    .transform((list) => [...new Set(list)].sort()),
  weekdays: weekdaysSchema.default([...ALL_WEEKDAYS]),
  amount: z.coerce.number().positive().catch(1).default(1),
  duration: durationSchema.default('ongoing'),
  notify: z.boolean().catch(true).default(true),
  starts_on: nullableText,
  ends_on: nullableText,
  is_active: z.boolean().catch(true).default(true),
  notes: nullableText,
});

/** One medication (GET /care/medications/{id}, list items, POST/PUT results). */
export const medicationSchema = z.preprocess(
  flattenMeta,
  medicationRowSchema.transform(
    (r): Medication => ({
      id: r.id,
      title: r.title,
      subtitle: r.subtitle,
      dose: r.dose,
      unit: r.unit,
      form: r.form,
      times: r.times,
      weekdays: r.weekdays,
      amount: r.amount,
      duration: r.duration,
      notify: r.notify,
      startsOn: r.starts_on,
      endsOn: r.ends_on,
      isActive: r.is_active,
      notes: r.notes,
    }),
  ),
);

export const medicationListSchema = z.array(medicationSchema);

const prepItemSchema = z.object({
  id: z.union([z.string(), z.number()]).transform(String),
  text: z.string(),
  done: z.boolean().catch(false).default(false),
});

const appointmentRowSchema = z.object({
  id: idSchema,
  title: z.string(),
  subtitle: nullableText,
  kind: kindSchema.default('in_person'),
  with: nullableText,
  specialty: nullableText,
  topic: topicSchema.default('other'),
  location: nullableText,
  remind_before: remindBeforeSchema.default('1d'),
  add_to_calendar: z.boolean().catch(false).default(false),
  // A malformed item drops the whole list rather than the whole appointment.
  prep: z.array(prepItemSchema).catch([]).default([]),
  status: statusSchema.default('scheduled'),
  scheduled_at: nullableText,
  is_active: z.boolean().catch(true).default(true),
  notes: nullableText,
});

/** One appointment (GET /care/appointments/{id}, list items, POST/PUT results). */
export const appointmentSchema = z.preprocess(
  flattenMeta,
  appointmentRowSchema.transform(
    (r): Appointment => ({
      id: r.id,
      title: r.title,
      subtitle: r.subtitle,
      kind: r.kind,
      withWhom: r.with,
      specialty: r.specialty,
      topic: r.topic,
      location: r.location,
      remindBefore: r.remind_before,
      addToCalendar: r.add_to_calendar,
      prep: r.prep,
      status: r.status,
      scheduledAt: r.scheduled_at,
      isActive: r.is_active,
      notes: r.notes,
    }),
  ),
);

export const appointmentListSchema = z.array(appointmentSchema);

const doseSchema = z.object({
  reminder_id: idSchema,
  title: z.string(),
  form: formSchema.default('tablet'),
  slot: slotSchema,
  taken: z.boolean().catch(false).default(false),
});

const nextAppointmentSchema = z.object({
  id: idSchema,
  kind: kindSchema.default('in_person'),
  title: z.string(),
  with: nullableText,
  scheduled_at: z.string(),
  days_until: z.coerce.number().int().catch(0).default(0),
  location: nullableText,
  remind_before: remindBeforeSchema.default('1d'),
});

/** GET /care/today. */
export const careTodaySchema = z
  .object({
    date: z.string(),
    doses: z.array(doseSchema).default([]),
    taken_count: z.coerce.number().int().optional(),
    total: z.coerce.number().int().optional(),
    // An unparseable appointment hides the card row instead of the whole card.
    next_appointment: nextAppointmentSchema.nullish().catch(null),
  })
  .transform((t): CareToday => {
    const doses = t.doses
      .map((d) => ({
        reminderId: d.reminder_id,
        title: d.title,
        form: d.form,
        slot: d.slot,
        taken: d.taken,
      }))
      .sort((a, b) => a.slot.localeCompare(b.slot));
    const na = t.next_appointment;
    return {
      date: t.date,
      doses,
      takenCount: t.taken_count ?? doses.filter((d) => d.taken).length,
      total: t.total ?? doses.length,
      nextAppointment: na
        ? {
            id: na.id,
            kind: na.kind,
            title: na.title,
            withWhom: na.with,
            scheduledAt: na.scheduled_at,
            daysUntil: na.days_until,
            location: na.location,
            remindBefore: na.remind_before,
          }
        : null,
    };
  });

/**
 * An enum list as `[{value,label}]` or `{value: label}`. Options whose value
 * this bundle doesn't know are dropped (a fallback would duplicate a real
 * option); `known` = null accepts any value (free-text units).
 */
function optionList<T extends string>(known: readonly T[] | null) {
  return z
    .unknown()
    .transform((raw): CareOption<T>[] => {
      const entries: Array<[unknown, unknown]> = Array.isArray(raw)
        ? raw.map((o) =>
            typeof o === 'object' && o !== null
              ? [(o as Record<string, unknown>).value, (o as Record<string, unknown>).label]
              : [o, o],
          )
        : typeof raw === 'object' && raw !== null
          ? Object.entries(raw)
          : [];
      const out: CareOption<T>[] = [];
      for (const [value, label] of entries) {
        if (typeof value !== 'string' || value === '') continue;
        if (known && !(known as readonly string[]).includes(value)) continue;
        out.push({ value: value as T, label: typeof label === 'string' ? label : value });
      }
      return out;
    });
}

/** GET /care/enums. */
export const careEnumsSchema = z
  .object({
    forms: optionList(MEDICATION_FORMS),
    units: optionList<string>(null),
    durations: optionList(MEDICATION_DURATIONS),
    kinds: optionList(APPOINTMENT_KINDS),
    topics: optionList(APPOINTMENT_TOPICS),
    remind_before: optionList(REMIND_BEFORE),
  })
  .transform(
    (e): CareEnums => ({
      forms: e.forms,
      units: e.units,
      durations: e.durations,
      kinds: e.kinds,
      topics: e.topics,
      remindBefore: e.remind_before,
    }),
  );
