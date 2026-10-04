import { z } from 'zod';

import {
  IVF_OUTCOMES,
  IVF_TWW_MOODS,
  type IvfDangerSign,
  type IvfLutealMed,
  type IvfNextStep,
  type IvfOutcome,
  type IvfTww,
} from '../model/tww';
import { IVF_ROUTES } from '../model/types';
import { ivfCycleSchema } from './schema';

/* `GET /ivf/tww`, `POST /ivf/cycles/current/outcome` and catalog `ivf_danger_signs` (CB-IVF-05). */

const nullableText = z.string().nullable().catch(null);
const nullableInt = z.number().int().nullable().catch(null);
const mood = z.enum(IVF_TWW_MOODS);

/** Keep the valid items of a list, never fail the list. */
function each<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((items) =>
      items.flatMap((raw) => {
        const item = schema.safeParse(raw);
        return item.success ? [item.data] : [];
      }),
    );
}

const lutealSchema = z
  .object({
    med_id: z.number().int(),
    name: z.string(),
    dose: z.union([z.string(), z.number()]).nullable().catch(null),
    unit: nullableText,
    route: z.enum(IVF_ROUTES).catch('other'),
    doses: each(z.object({ slot: z.string().regex(/^\d{2}:\d{2}$/), taken: z.boolean().catch(false) })),
    taken: z.number().int().min(0).catch(0),
    total: z.number().int().min(0).catch(0),
  })
  .transform(
    (d): IvfLutealMed => ({
      medId: d.med_id,
      name: d.name,
      dose: d.dose === null ? null : String(d.dose),
      unit: d.unit,
      route: d.route,
      slots: d.doses,
      taken: d.taken,
      total: d.total,
    }),
  );

export const ivfTwwSchema = z
  .object({
    cycle: ivfCycleSchema.nullable().catch(null),
    transfer_on: nullableText,
    beta_on: nullableText,
    days_since_transfer: nullableInt,
    days_to_beta: nullableInt,
    today: z.object({ date: z.string(), mood: mood.nullable().catch(null) }),
    moods: each(z.object({ date: z.string(), mood })),
    luteal_support: each(lutealSchema),
  })
  .transform(
    (d): IvfTww => ({
      cycle: d.cycle,
      transferOn: d.transfer_on,
      betaOn: d.beta_on,
      daysSinceTransfer: d.days_since_transfer,
      daysToBeta: d.days_to_beta,
      today: d.today.date,
      todayMood: d.today.mood,
      moods: d.moods,
      lutealSupport: d.luteal_support,
    }),
  );

const NEXT_STEPS = ['pregnancy_setup', 'loss', 'new_cycle'] as const;

export const ivfOutcomeSchema = z
  .object({
    cycle: z.object({ outcome: z.enum(IVF_OUTCOMES), outcome_on: nullableText }),
    next_steps: each(z.enum(NEXT_STEPS)),
  })
  .transform(
    (d): IvfOutcome => ({
      result: d.cycle.outcome,
      outcomeOn: d.cycle.outcome_on,
      nextSteps: d.next_steps as IvfNextStep[],
    }),
  );

const text = z.string().trim().min(1).nullable().catch(null);

export const ivfDangerSignsSchema = z
  .object({
    items: each(
      z
        .object({
          code: z.string().min(1),
          title: text,
          body: text,
          meta: z.record(z.string(), z.unknown()).nullable().catch(null),
        })
        .transform((d): IvfDangerSign => {
          const lines = z.array(z.string().regex(/^\d{2,6}$/)).safeParse(d.meta?.hotlines);
          return { code: d.code, title: d.title, body: d.body, hotlines: lines.success ? lines.data : [] };
        }),
    ),
  })
  .catch({ items: [] })
  .transform((g) => g.items);
