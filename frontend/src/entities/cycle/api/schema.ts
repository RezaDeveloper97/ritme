import { z } from 'zod';

import type { CycleCalculation, MonthSummary } from '../model/types';

/**
 * Validate the API's `CycleCalculation` at the boundary (CLAUDE.md §10) and map
 * its snake_case fields onto our camelCase domain shape. We keep only the fields
 * the app renders; the many internal scoring factors the backend also sends
 * (`age_factor`, `symptom_score`, …) are intentionally dropped — they're not
 * shown to users and must not leak into logs or state (§11).
 */
export const cycleCalculationSchema = z
  .object({
    cycle_day: z.number(),
    phase: z.string(),
    subphase: z.string().nullable().default(null),
    estimated_ovulation_day: z.number(),
    cycle_length_used: z.number(),
    is_fertile_window: z.boolean().default(false),
    is_pms_window: z.boolean().default(false),
    is_period_tomorrow: z.boolean().default(false),
    final_probability: z.number().default(0),
    cycle_variability: z.string().nullable().default(null),
  })
  .transform(
    (c): CycleCalculation => ({
      cycleDay: c.cycle_day,
      phase: c.phase,
      subphase: c.subphase,
      estimatedOvulationDay: c.estimated_ovulation_day,
      cycleLength: c.cycle_length_used,
      isFertileWindow: c.is_fertile_window,
      isPmsWindow: c.is_pms_window,
      isPeriodTomorrow: c.is_period_tomorrow,
      fertilityPercent: c.final_probability,
      cycleVariability: c.cycle_variability,
    }),
  );

/** `{ calculation, calculation_status, is_recalculating }` — today / by-date. */
export const cycleCalculationEnvelopeSchema = z
  .object({
    calculation: cycleCalculationSchema.nullable().default(null),
    calculation_status: z.string().default('completed'),
    is_recalculating: z.boolean().default(false),
  })
  .transform((e) => ({
    calculation: e.calculation,
    calculationStatus: e.calculation_status,
    isRecalculating: e.is_recalculating,
  }));

const monthSummarySchema = z
  .object({
    fertile_days: z.number().default(0),
    period_days: z.number().default(0),
    pms_days: z.number().default(0),
  })
  .transform(
    (s): MonthSummary => ({
      fertileDays: s.fertile_days,
      periodDays: s.period_days,
      pmsDays: s.pms_days,
    }),
  );

/** `{ calculations[], month_summary, … }` — a month of calculations. */
export const cycleMonthEnvelopeSchema = z
  .object({
    calculations: z.array(cycleCalculationSchema).default([]),
    calculation_status: z.string().default('completed'),
    is_recalculating: z.boolean().default(false),
    month_summary: monthSummarySchema.optional(),
  })
  .transform((e) => ({
    calculations: e.calculations,
    calculationStatus: e.calculation_status,
    isRecalculating: e.is_recalculating,
    monthSummary: e.month_summary ?? null,
  }));

/** `{ status, is_processing, version, … }` — calculation processing status. */
export const cycleStatusSchema = z.object({
  status: z.string().default('completed'),
  status_label: z.string().default(''),
  is_processing: z.boolean().default(false),
  version: z.number().default(1),
});
