import { z } from 'zod';

import {
  HOT_FLASH_SEVERITIES,
  HOT_FLASH_TRIGGERS,
  MENOPAUSE_STAGE_ANSWERS,
  type HotFlashTrigger,
  type MenopauseCatalogItem,
  type MenopauseFlash,
  type MenopauseMessage,
  type MenopauseProfile,
  type MenopauseToday,
} from '../model/types';

/*
 * Parsers of `/api/v1/menopause/*` (CB-MENO-02) and `/messages/menopause`
 * (CB-MENO-12). Lenient on optional fields (`.catch`) so one odd value never
 * blanks the home; strict on what a screen can't do without.
 */

const nullableText = z.string().nullable().catch(null);
const nullableBool = z.boolean().nullable().catch(null);
const nullableInt = z.number().int().nullable().catch(null);
const nullableDate = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/)
  .nullable()
  .catch(null);

export const catalogItemSchema = z
  .object({
    code: z.string(),
    title: nullableText,
    body: nullableText,
    needs_review: z.boolean().catch(true),
  })
  .transform((d): MenopauseCatalogItem => ({ code: d.code, title: d.title, body: d.body, needsReview: d.needs_review }));

const nullableCatalogItem = catalogItemSchema.nullable().catch(null);

export const menopauseProfileSchema = z
  .object({
    stage: z.enum(['peri', 'meno', 'post']).nullable().catch(null),
    stored_stage: z.enum(MENOPAUSE_STAGE_ANSWERS).nullable().catch(null),
    last_period: nullableDate,
    surgical: nullableBool,
    hrt: nullableBool,
    months_without_period: nullableInt,
    suggested_stage: z.literal('meno').nullable().catch(null),
    needs_stage: z.boolean().catch(false),
    post_menopausal: z.boolean().catch(false),
    tip: nullableCatalogItem,
  })
  .transform(
    (d): MenopauseProfile => ({
      stage: d.stage,
      storedStage: d.stored_stage,
      lastPeriod: d.last_period,
      surgical: d.surgical,
      hrt: d.hrt,
      monthsWithoutPeriod: d.months_without_period,
      suggestedStage: d.suggested_stage,
      needsStage: d.needs_stage,
      postMenopausal: d.post_menopausal,
      tip: d.tip,
    }),
  );

export const menopauseFlashSchema = z
  .object({
    id: z.number().int(),
    started_at: z.string(),
    running: z.boolean(),
    duration_s: nullableInt,
    elapsed_s: z.number().int().catch(0),
    severity: z.enum(HOT_FLASH_SEVERITIES).nullable().catch(null),
    night: z.boolean().catch(false),
    sweat: z.boolean().catch(false),
    triggers: z
      .array(z.unknown())
      .catch([])
      .transform((codes) => codes.filter((c): c is HotFlashTrigger => (HOT_FLASH_TRIGGERS as readonly unknown[]).includes(c))),
  })
  .transform(
    (d): MenopauseFlash => ({
      id: d.id,
      startedAt: d.started_at,
      running: d.running,
      durationS: d.duration_s,
      elapsedS: d.elapsed_s,
      severity: d.severity,
      night: d.night,
      sweat: d.sweat,
      triggers: d.triggers,
    }),
  );

const scoreSchema = z.object({
  total: z.number().int(),
  max: z.number().int(),
  band: z
    .object({ code: z.string(), title: nullableText })
    .nullable()
    .catch(null),
  delta: nullableInt,
});

const trendPointSchema = z.object({
  month: z.string(),
  total: nullableInt,
  band: nullableText,
});

const treatmentSchema = z
  .object({
    id: z.number().int(),
    kind: z.enum(['hrt', 'supplement', 'lifestyle']),
    name: z.string(),
    taken_today: z.boolean().catch(false),
    days_taken: z.number().int().catch(0),
    days: z.number().int().catch(7),
    weekly_goal: nullableInt.optional(),
    goal_unit: nullableText.optional(),
    amount: z.number().int().catch(0).optional(),
    review_on: nullableDate,
  })
  .transform((d) => ({
    id: d.id,
    kind: d.kind,
    name: d.name,
    takenToday: d.taken_today,
    daysTaken: d.days_taken,
    days: d.days,
    // weekly goals (lifestyle minutes / sessions) report this week's amount against the goal instead of days
    goal: d.weekly_goal != null ? { target: d.weekly_goal, unit: d.goal_unit ?? 'sessions', amount: d.amount ?? 0 } : null,
    reviewOn: d.review_on,
  }));

/** Drops the entries that fail to parse instead of failing the whole list. */
function lenientArray<T extends z.ZodTypeAny>(item: T) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((rows) =>
      rows.flatMap((row) => {
        const parsed = item.safeParse(row);
        return parsed.success ? [parsed.data as z.output<T>] : [];
      }),
    );
}

export const menopauseTodaySchema = z
  .object({
    date: z.string(),
    profile: menopauseProfileSchema,
    hot_flashes: z.object({
      count: z.number().int().catch(0),
      night_count: z.number().int().catch(0),
      running: menopauseFlashSchema.nullable().catch(null),
    }),
    night_sweats: z.object({ count: z.number().int().catch(0) }).catch({ count: 0 }),
    sleep: z.object({ hours: z.number() }).nullable().catch(null),
    score: z.object({
      max: z.number().int().catch(44),
      latest: scoreSchema.nullable().catch(null),
      trend: lenientArray(trendPointSchema),
    }),
    checkups: z
      .array(z.record(z.string(), z.unknown()))
      .catch([]),
    treatment: lenientArray(treatmentSchema),
    bleeding: z
      .object({
        alert: z.boolean().catch(false),
        last_on: nullableDate,
        alert_item: nullableCatalogItem,
      })
      .catch({ alert: false, last_on: null, alert_item: null }),
  })
  .transform(
    (d): MenopauseToday => ({
      date: d.date,
      profile: d.profile,
      hotFlashes: { count: d.hot_flashes.count, nightCount: d.hot_flashes.night_count, running: d.hot_flashes.running },
      nightSweats: { count: d.night_sweats.count },
      sleep: d.sleep ? { hours: d.sleep.hours } : null,
      score: { max: d.score.max, latest: d.score.latest, trend: d.score.trend },
      checkups: d.checkups,
      treatment: d.treatment,
      bleeding: { alert: d.bleeding.alert, lastOn: d.bleeding.last_on, alertItem: d.bleeding.alert_item },
    }),
  );

const messageSchema = z
  .object({
    key: z.string(),
    kind: z.enum(['alert', 'reminder', 'tip']),
    priority: z.enum(['high', 'medium', 'low']).catch('low'),
    title: nullableText,
    body: nullableText,
    action: nullableText,
    link: nullableText,
  })
  .transform(
    (d): MenopauseMessage => ({
      key: d.key,
      kind: d.kind,
      priority: d.priority,
      title: d.title,
      body: d.body,
      action: d.action,
      link: d.link,
    }),
  );

export const menopauseMessagesSchema = z
  .object({ messages: lenientArray(messageSchema) })
  .transform((d): MenopauseMessage[] => d.messages);
