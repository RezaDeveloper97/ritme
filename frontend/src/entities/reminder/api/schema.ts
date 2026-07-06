import { z } from 'zod';

import type { Reminder, ReminderEnums } from '../model/types';

const reminderTypeSchema = z
  .enum(['doctor', 'medication', 'appointment', 'custom'])
  .catch('custom');

const recurrenceSchema = z.enum(['none', 'daily', 'weekly', 'monthly']).catch('none');

/** The API may send "16:00" or "16:00:00" — normalize to "HH:MM". */
const timeSchema = z
  .string()
  .nullable()
  .default(null)
  .transform((v) => (v ? v.slice(0, 5) : null));

/** Boundary parser for a single reminder (CLAUDE.md §10 — zod at the boundary). */
export const reminderSchema = z
  .object({
    id: z.union([z.number(), z.string()]).transform(String),
    type: reminderTypeSchema,
    title: z.string(),
    subtitle: z.string().nullable().default(null),
    notes: z.string().nullable().default(null),
    scheduled_at: z.string().nullable().default(null),
    recurrence: recurrenceSchema,
    recurrence_time: timeSchema,
    starts_on: z.string().nullable().default(null),
    ends_on: z.string().nullable().default(null),
    is_active: z.boolean().default(true),
  })
  .transform(
    (r): Reminder => ({
      id: r.id,
      type: r.type,
      title: r.title,
      subtitle: r.subtitle,
      notes: r.notes,
      scheduledAt: r.scheduled_at,
      recurrence: r.recurrence,
      recurrenceTime: r.recurrence_time,
      startsOn: r.starts_on,
      endsOn: r.ends_on,
      isActive: r.is_active,
    }),
  );

/** Boundary parser for GET /reminders. */
export const reminderListSchema = z.array(reminderSchema);

/** Boundary parser for GET /reminders/enums. */
export const reminderEnumsSchema = z
  .object({
    types: z
      .array(
        z.object({
          value: reminderTypeSchema,
          label: z.string().default(''),
          icon: z.string().default(''),
        }),
      )
      .default([]),
    recurrences: z
      .array(
        z.object({
          value: recurrenceSchema,
          label: z.string().default(''),
        }),
      )
      .default([]),
  })
  .transform(
    (e): ReminderEnums => ({
      types: e.types,
      recurrences: e.recurrences,
    }),
  );
