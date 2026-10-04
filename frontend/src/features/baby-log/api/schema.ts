import { z } from 'zod';

import {
  DIAPER_KINDS,
  FEED_SIDES,
  FEED_TYPES,
  type BabyDiaper,
  type BabyFeed,
  type BabyLogDay,
  type BabyLogSummary,
  type BabySleep,
  type BabyToday,
  type DiaperDay,
  type DiaperTotals,
  type FeedDay,
  type FeedTotals,
  type SleepDay,
  type SleepTotals,
} from '../model/types';

/*
 * Parsers of `/api/v1/children/{id}/feeds|sleeps|diapers|baby-logs` (B-N5-03).
 * Strict on identity and timestamps (the timer can't run without them),
 * lenient (`.catch`) on totals so one odd number never blanks the screen.
 */

const text = z.string().nullable().catch(null);
const count = z.number().catch(0);
const stamp = z.string().regex(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/);
const side = z.enum(FEED_SIDES).nullable().catch(null);

export const feedSchema = z
  .object({
    id: z.number().int(),
    type: z.enum(FEED_TYPES),
    started_at: stamp,
    ended_at: stamp.nullable().catch(null),
    is_active: z.boolean(),
    active_side: side,
    side_started_at: stamp.nullable().catch(null),
    last_side: side,
    left_seconds: count,
    right_seconds: count,
    duration_seconds: count,
    amount_ml: z.number().nullable().catch(null),
    note: text,
  })
  .transform(
    (d): BabyFeed => ({
      id: d.id,
      type: d.type,
      startedAt: d.started_at,
      endedAt: d.ended_at,
      isActive: d.is_active,
      activeSide: d.active_side,
      sideStartedAt: d.side_started_at,
      lastSide: d.last_side,
      leftSeconds: d.left_seconds,
      rightSeconds: d.right_seconds,
      durationSeconds: d.duration_seconds,
      amountMl: d.amount_ml,
      note: d.note,
    }),
  );

const feedTotalsSchema = z
  .object({
    count,
    breast: count,
    bottle: count,
    pump: count,
    left_seconds: count,
    right_seconds: count,
    total_seconds: count,
    bottle_ml: count,
    pump_ml: count,
  })
  .transform(
    (d): FeedTotals => ({
      count: d.count,
      breast: d.breast,
      bottle: d.bottle,
      pump: d.pump,
      leftSeconds: d.left_seconds,
      rightSeconds: d.right_seconds,
      totalSeconds: d.total_seconds,
      bottleMl: d.bottle_ml,
      pumpMl: d.pump_ml,
    }),
  );

const EMPTY_FEEDS: FeedTotals = {
  count: 0,
  breast: 0,
  bottle: 0,
  pump: 0,
  leftSeconds: 0,
  rightSeconds: 0,
  totalSeconds: 0,
  bottleMl: 0,
  pumpMl: 0,
};

/** Items lists drop a row that doesn't parse instead of failing the day. */
function lenientList<T>(item: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((list) =>
      list.flatMap((raw) => {
        const parsed = item.safeParse(raw);
        return parsed.success ? [parsed.data] : [];
      }),
    );
}

export const feedDaySchema = z
  .object({
    date: z.string(),
    active: feedSchema.nullable().catch(null),
    last: feedSchema.nullable().catch(null),
    next_side: side,
    summary: feedTotalsSchema.catch(EMPTY_FEEDS),
    items: lenientList(feedSchema),
  })
  .transform(
    (d): FeedDay => ({
      date: d.date,
      active: d.active,
      last: d.last,
      nextSide: d.next_side,
      summary: d.summary,
      items: d.items,
    }),
  );

export const sleepSchema = z
  .object({
    id: z.number().int(),
    started_at: stamp,
    ended_at: stamp.nullable().catch(null),
    is_active: z.boolean(),
    duration_seconds: count,
    note: text,
  })
  .transform(
    (d): BabySleep => ({
      id: d.id,
      startedAt: d.started_at,
      endedAt: d.ended_at,
      isActive: d.is_active,
      durationSeconds: d.duration_seconds,
      note: d.note,
    }),
  );

const EMPTY_SLEEP: SleepTotals = { count: 0, seconds: 0, longestSeconds: 0 };

const sleepTotalsSchema = z
  .object({ count, seconds: count, longest_seconds: count })
  .transform((d): SleepTotals => ({ count: d.count, seconds: d.seconds, longestSeconds: d.longest_seconds }));

export const sleepDaySchema = z
  .object({
    date: z.string(),
    active: sleepSchema.nullable().catch(null),
    summary: sleepTotalsSchema.catch(EMPTY_SLEEP),
    items: lenientList(sleepSchema),
  })
  .transform((d): SleepDay => d);

export const diaperSchema = z
  .object({ id: z.number().int(), changed_at: stamp, kind: z.enum(DIAPER_KINDS), note: text })
  .transform((d): BabyDiaper => ({ id: d.id, changedAt: d.changed_at, kind: d.kind, note: d.note }));

const EMPTY_DIAPERS: DiaperTotals = { count: 0, wet: 0, dirty: 0, both: 0 };

const diaperTotalsSchema = z
  .object({ count, wet: count, dirty: count, both: count })
  .transform((d): DiaperTotals => d);

export const diaperDaySchema = z
  .object({
    date: z.string(),
    summary: diaperTotalsSchema.catch(EMPTY_DIAPERS),
    items: lenientList(diaperSchema),
  })
  .transform((d): DiaperDay => d);

const logDaySchema = z
  .object({
    date: z.string(),
    feeds: feedTotalsSchema.catch(EMPTY_FEEDS),
    sleep: sleepTotalsSchema.catch(EMPTY_SLEEP),
    diapers: diaperTotalsSchema.catch(EMPTY_DIAPERS),
  })
  .transform((d): BabyLogDay => d);

export const babyTodaySchema = z
  .object({
    date: z.string(),
    feeds: z
      .object({ last: feedSchema.nullable().catch(null) })
      .passthrough()
      .catch({ last: null }),
    sleep: sleepTotalsSchema.catch(EMPTY_SLEEP),
    diapers: diaperTotalsSchema.catch(EMPTY_DIAPERS),
    feeding_now: z.boolean().catch(false),
    sleeping_now: z.boolean().catch(false),
  })
  .transform(
    (d): BabyToday => ({
      date: d.date,
      feeds: feedTotalsSchema.catch(EMPTY_FEEDS).parse(d.feeds),
      lastFeed: d.feeds.last,
      sleep: d.sleep,
      diapers: d.diapers,
      feedingNow: d.feeding_now,
      sleepingNow: d.sleeping_now,
    }),
  );

/** `ChildHome.today` (typed `unknown` by the child entity) → the «امروز» card, null when absent or malformed. */
export function parseBabyToday(raw: unknown): BabyToday | null {
  if (raw === null || raw === undefined) return null;
  const parsed = babyTodaySchema.safeParse(raw);
  return parsed.success ? parsed.data : null;
}

const percent = z.number().nullable().catch(null);

export const babyLogSummarySchema = z
  .object({
    from: z.string(),
    to: z.string(),
    today: babyTodaySchema,
    days: lenientList(logDaySchema),
    averages: z
      .object({
        feeds_per_day: count,
        sleep_seconds_per_day: count,
        diapers_per_day: count,
        left_percent: percent,
        right_percent: percent,
      })
      .catch({ feeds_per_day: 0, sleep_seconds_per_day: 0, diapers_per_day: 0, left_percent: null, right_percent: null }),
  })
  .transform(
    (d): BabyLogSummary => ({
      from: d.from,
      to: d.to,
      today: d.today,
      days: d.days,
      averages: {
        feedsPerDay: d.averages.feeds_per_day,
        sleepSecondsPerDay: d.averages.sleep_seconds_per_day,
        diapersPerDay: d.averages.diapers_per_day,
        leftPercent: d.averages.left_percent,
        rightPercent: d.averages.right_percent,
      },
    }),
  );
