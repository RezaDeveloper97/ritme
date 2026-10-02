import { z } from 'zod';

import {
  LOSS_MOODS,
  LOSS_NEXT_STEPS,
  LOSS_TYPES,
  type LossAppointment,
  type LossCatalogItem,
  type LossNote,
  type LossState,
} from '../model/types';

/*
 * Zod parsers of the `/api/v1/loss` payloads (backend-go/contract/golden/loss)
 * and the `loss_*` catalog groups. Lenient where the API may grow: an unknown
 * enum value degrades (type → unspecified, mood/next step → null) rather than
 * failing the whole screen.
 */

const date = z.string().regex(/^\d{4}-\d{2}-\d{2}$/).nullable().catch(null);
const str = z.string().nullable().catch(null);

const appointmentSchema = z
  .object({ id: z.number(), scheduled_at: z.string() })
  .transform((a): LossAppointment => ({ id: a.id, scheduledAt: a.scheduled_at }))
  .nullable()
  .catch(null);

const lossSchema = z.object({
  id: z.number(),
  type: z.enum(LOSS_TYPES).catch('unspecified'),
  occurred_on: date,
  notify_companion: z.boolean().catch(false),
  companion_notified: z.boolean().catch(false),
  content_stopped: z.boolean().catch(true),
  next_step: z.enum(LOSS_NEXT_STEPS).nullable().catch(null),
  created_at: str,
});

const followupSchema = z.object({
  bleeding: z.object({ stopped: z.boolean().catch(false), stopped_on: date, today: str }),
  beta: z.object({
    negative: z.boolean().catch(false),
    negative_on: date,
    next_on: date,
    appointment: appointmentSchema,
  }),
  visit: z.object({ suggested_on: date, appointment: appointmentSchema }),
});

const moodEntry = z.object({ date: z.string(), mood: z.enum(LOSS_MOODS) });

export const lossStateSchema = z
  .object({
    loss: lossSchema.nullable().catch(null),
    losses_count: z.number().catch(0),
    recurrent_hint: z.boolean().catch(false),
    followup: followupSchema.nullable().catch(null),
    mood: z
      .object({
        today: z.enum(LOSS_MOODS).nullable().catch(null),
        recent: z.array(z.unknown()).catch([]),
      })
      .nullable()
      .catch(null),
    note: z
      .object({ has_note: z.boolean().catch(false), updated_at: str })
      .nullable()
      .catch(null),
  })
  .transform(
    (d): LossState => ({
      loss: d.loss
        ? {
            id: d.loss.id,
            type: d.loss.type,
            occurredOn: d.loss.occurred_on,
            notifyCompanion: d.loss.notify_companion,
            companionNotified: d.loss.companion_notified,
            contentStopped: d.loss.content_stopped,
            nextStep: d.loss.next_step,
            createdAt: d.loss.created_at,
          }
        : null,
      lossesCount: d.losses_count,
      recurrentHint: d.recurrent_hint,
      followup: d.followup
        ? {
            bleeding: {
              stopped: d.followup.bleeding.stopped,
              stoppedOn: d.followup.bleeding.stopped_on,
              today: d.followup.bleeding.today,
            },
            beta: {
              negative: d.followup.beta.negative,
              negativeOn: d.followup.beta.negative_on,
              nextOn: d.followup.beta.next_on,
              appointment: d.followup.beta.appointment,
            },
            visit: { suggestedOn: d.followup.visit.suggested_on, appointment: d.followup.visit.appointment },
          }
        : null,
      mood: d.mood
        ? {
            today: d.mood.today,
            recent: d.mood.recent.flatMap((raw) => {
              const parsed = moodEntry.safeParse(raw);
              return parsed.success ? [parsed.data] : [];
            }),
          }
        : null,
      note: d.note ? { hasNote: d.note.has_note, updatedAt: d.note.updated_at } : null,
    }),
  );

export const lossNoteSchema = z
  .object({ note: str, updated_at: str })
  .transform((n): LossNote => ({ note: n.note, updatedAt: n.updated_at }));

const text = z.string().trim().min(1).nullable().catch(null);

const catalogItemSchema = z
  .object({
    code: z.string(),
    title: text,
    body: text,
    meta: z.record(z.string(), z.unknown()).nullable().catch(null),
  })
  .transform((i): LossCatalogItem => ({ code: i.code, title: i.title, body: i.body, meta: i.meta }));

/** `GET /catalog/<loss group>` → `data`; a malformed item is dropped, never the list. */
export const lossCatalogSchema = z
  .object({ items: z.array(z.unknown()).catch([]) })
  .catch({ items: [] })
  .transform((g): LossCatalogItem[] =>
    g.items.flatMap((raw) => {
      const parsed = catalogItemSchema.safeParse(raw);
      return parsed.success ? [parsed.data] : [];
    }),
  );
