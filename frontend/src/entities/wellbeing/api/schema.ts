import { z } from 'zod';

import type { WeeklyWellbeing, WellbeingMetric, WellbeingMetricKey } from '../model/types';

/** Order the tiles render in, regardless of what order the API lists them. */
const METRIC_ORDER: WellbeingMetricKey[] = ['mood', 'sleep', 'energy'];

const percentSchema = z.number().nullable().catch(null).default(null);

const metricSchema = z
  .object({
    key: z.enum(['mood', 'sleep', 'energy']),
    percent: percentSchema,
    previous_percent: percentSchema,
    delta: percentSchema,
  })
  .transform(
    (m): WellbeingMetric => ({
      key: m.key,
      percent: m.percent,
      previousPercent: m.previous_percent,
      delta: m.delta,
    }),
  );

/**
 * `data.section.data` of `GET /home/sections/weekly_summary` — validated at the
 * API boundary (§10) and mapped onto our camelCase domain shape. Labels come
 * from the app's own messages (§6), so the localized ones the API sends are
 * ignored here.
 */
export const weeklyWellbeingSchema = z
  .object({
    items: z.array(metricSchema.nullable().catch(null)).default([]),
    range: z.object({ from: z.string(), to: z.string() }),
    logged_days: z.number().default(0),
    previous_logged_days: z.number().default(0),
    overall_percent: percentSchema,
  })
  .transform((s): WeeklyWellbeing => {
    const byKey = new Map(s.items.filter((m): m is WellbeingMetric => m !== null).map((m) => [m.key, m]));

    return {
      metrics: METRIC_ORDER.map(
        (key) => byKey.get(key) ?? { key, percent: null, previousPercent: null, delta: null },
      ),
      from: s.range.from,
      to: s.range.to,
      loggedDays: s.logged_days,
      previousLoggedDays: s.previous_logged_days,
      overallPercent: s.overall_percent,
    };
  });

/**
 * The section envelope. A section the backend chose not to build maps to
 * `null`, which the card renders as its "nothing logged yet" state.
 */
export const weeklyWellbeingSectionSchema = z
  .object({
    section: z.object({ data: weeklyWellbeingSchema }).nullable().default(null),
  })
  .transform((e): WeeklyWellbeing | null => e.section?.data ?? null);
