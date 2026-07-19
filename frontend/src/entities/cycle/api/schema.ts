import { z } from 'zod';

import type {
  CalculatedValues,
  CardAction,
  ConfidenceLevel,
  CycleAnchors,
  CycleCalculation,
  CycleDataQuality,
  CycleEngineMetrics,
  CycleFertilityLevel,
  CycleForecast,
  CycleValueLayer,
  CycleView,
  DailyCard,
  DataQualityLevel,
  DataStatus,
  EffectiveValues,
  MainPhase,
  MonthSummary,
  ResolutionSource,
} from '../model/types';

/**
 * Validate the API's `CycleCalculation` at the boundary (CLAUDE.md §10) and map
 * its snake_case fields onto our camelCase domain shape. We keep only the fields
 * the app renders; the many internal scoring factors the backend also sends
 * (`age_factor`, `symptom_score`, …) are intentionally dropped — they're not
 * shown to users and must not leak into logs or state (§11).
 */
export const cycleCalculationSchema = z
  .object({
    // Gregorian `YYYY-MM-DD` the backend stamps on every calculation. The
    // calendar keys each day cell by this so a filtered/sparse month array still
    // maps to the right dates (§7 — converted to Jalali only for display).
    calculation_date: z.string(),
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
      calculationDate: c.calculation_date,
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

/**
 * The backend always sends a calculation *object*; when the profile is
 * incomplete (no `last_period_start`) it returns an "empty" one with null core
 * fields plus an `incomplete_profile` text flag (see the API's
 * `HealthDataEngine::getEmptyCalculation`). Collapse that sentinel to `null`
 * here so the boundary parser (§10) doesn't throw on the null fields — every
 * consumer already treats a null `calculation` as "no data yet" and shows the
 * onboarding / "unavailable" state. Any other malformed shape still throws.
 */
const isEmptyCalculation = (v: unknown): boolean =>
  !!v && typeof v === 'object' && (v as { cycle_day?: unknown }).cycle_day == null;

const nullableCalculationSchema = z.preprocess(
  (v) => (isEmptyCalculation(v) ? null : v),
  cycleCalculationSchema.nullable(),
);

// --- Rich daily payload (spec §19), served under `cycle_view` -----------------
// Enum-valued fields are parsed as strings and cast to their union types: an
// unexpected new backend value should degrade gracefully, not throw the whole
// envelope (which would blank the screen). Health copy stays out of logs (§11).

const cardActionSchema = z
  .object({ type: z.string(), label: z.string() })
  .transform((a): CardAction => ({ type: a.type, label: a.label }));

const dailyCardSchema = z
  .object({
    title: z.string(),
    subtitle: z.string().default(''),
    data_status: z.string(),
    // Coerce unknown enum values to a safe default at the boundary rather than
    // letting them reach the renderer — belt-and-braces with the component fallback.
    fertility_level: z
      .enum(['very_low', 'low', 'medium', 'high', 'very_high'])
      .catch('low'),
    fertility_label: z.string().default(''),
    badges: z.array(z.string()).default([]),
    primary_action: cardActionSchema.nullable().default(null),
    secondary_actions: z.array(cardActionSchema).default([]),
  })
  .transform(
    (c): DailyCard => ({
      title: c.title,
      subtitle: c.subtitle,
      dataStatus: c.data_status as DataStatus,
      fertilityLevel: c.fertility_level,
      fertilityLabel: c.fertility_label,
      badges: c.badges,
      primaryAction: c.primary_action,
      secondaryActions: c.secondary_actions,
    }),
  );

const forecastSchema = z
  .object({
    next_period_start: z.string(),
    next_period_end: z.string(),
    estimated_ovulation_date: z.string(),
    fertile_window_start: z.string(),
    fertile_window_end: z.string(),
    source: z.string().default('profile'),
    confidence: z.string().default('low'),
    confidence_reasons: z.array(z.string()).default([]),
  })
  .transform(
    (p): CycleForecast => ({
      nextPeriodStart: p.next_period_start,
      nextPeriodEnd: p.next_period_end,
      estimatedOvulationDate: p.estimated_ovulation_date,
      fertileWindowStart: p.fertile_window_start,
      fertileWindowEnd: p.fertile_window_end,
      source: p.source,
      confidence: p.confidence as ConfidenceLevel,
      confidenceReasons: p.confidence_reasons,
    }),
  );

const valueLayerFields = {
  cycle_length: z.number().nullable().default(null),
  period_duration: z.number().nullable().default(null),
};

const profileValuesSchema = z
  .object(valueLayerFields)
  .transform(
    (v): CycleValueLayer => ({
      cycleLength: v.cycle_length,
      periodDuration: v.period_duration,
    }),
  );

const calculatedValuesSchema = z
  .object({ ...valueLayerFields, based_on_cycles: z.number().nullable().default(null) })
  .transform(
    (v): CalculatedValues => ({
      cycleLength: v.cycle_length,
      periodDuration: v.period_duration,
      basedOnCycles: v.based_on_cycles,
    }),
  );

const effectiveValuesSchema = z
  .object({
    ...valueLayerFields,
    source: z.string().default('default'),
    period_duration_source: z.string().default('default'),
  })
  .transform(
    (v): EffectiveValues => ({
      cycleLength: v.cycle_length,
      periodDuration: v.period_duration,
      source: v.source,
      periodDurationSource: v.period_duration_source,
    }),
  );

const dataQualityDetailsSchema = z
  .object({
    confidence: z.string().default('low'),
    confidence_reasons: z.array(z.string()).default([]),
    regularity_status: z.string().default('not_enough_data'),
    is_irregular_possible: z.boolean().default(false),
    missing_period_end: z.boolean().default(false),
  })
  .transform(
    (q): CycleDataQuality => ({
      confidence: q.confidence as ConfidenceLevel,
      confidenceReasons: q.confidence_reasons,
      regularityStatus: q.regularity_status,
      isIrregularPossible: q.is_irregular_possible,
      missingPeriodEnd: q.missing_period_end,
    }),
  );

/** The v1.1 anchors block (task.md §10, §35): current period + sources + predictions. */
const anchorsSchema = z
  .object({
    current_period_start: z.string().nullable().default(null),
    current_period_start_source: z.string().default('unknown'),
    current_period_end: z.string().nullable().default(null),
    current_period_end_source: z.string().default('unknown'),
    current_period_end_is_confirmed: z.boolean().default(false),
    predicted_next_period_start: z.string().nullable().default(null),
    estimated_ovulation_date: z.string().nullable().default(null),
  })
  .transform(
    (a): CycleAnchors => ({
      currentPeriodStart: a.current_period_start,
      currentPeriodStartSource: a.current_period_start_source,
      currentPeriodEnd: a.current_period_end,
      currentPeriodEndSource: a.current_period_end_source,
      currentPeriodEndIsConfirmed: a.current_period_end_is_confirmed,
      predictedNextPeriodStart: a.predicted_next_period_start,
      estimatedOvulationDate: a.estimated_ovulation_date,
    }),
  );

/** The v1.1 effective-metrics block (task.md §9, §27, §35). */
const engineMetricsSchema = z
  .object({
    effective_cycle_length: z.number(),
    effective_period_length: z.number(),
    cycle_variability: z.number().nullable().default(null),
  })
  .transform(
    (m): CycleEngineMetrics => ({
      effectiveCycleLength: m.effective_cycle_length,
      effectivePeriodLength: m.effective_period_length,
      cycleVariability: m.cycle_variability,
    }),
  );

/**
 * The `cycle_view` object: the v1.1 engine status (task.md §35) merged with the
 * legacy render keys (daily card, forecast, three-layer values). New fields are
 * optional-with-defaults so an older backend still parses.
 */
export const cycleViewSchema = z
  .object({
    date: z.string(),
    cycle_day: z.number().nullable().default(null),
    phase: z.string().nullable().default(null),
    subphase: z.string().nullable().default(null),
    main_phase: z.string().nullable().default(null),
    fertility_level: z.string().nullable().default(null),
    data_status: z.string().nullable().default(null),
    days_to_ovulation: z.number().nullable().default(null),
    days_to_period: z.number().nullable().default(null),
    days_late: z.number().nullable().default(null),
    anchors: anchorsSchema.nullable().default(null),
    metrics: engineMetricsSchema.nullable().default(null),
    confidence: z.string().nullable().default(null),
    confidence_reasons: z.array(z.string()).default([]),
    data_quality: z.string().nullable().default(null),
    data_quality_details: dataQualityDetailsSchema.nullable().default(null),
    resolution_source: z.string().nullable().default(null),
    is_predicted: z.boolean().default(false),
    requires_user_input: z.boolean().default(false),
    warnings: z.array(z.string()).default([]),
    predictions: forecastSchema.nullable().default(null),
    profile_values: profileValuesSchema,
    calculated_values: calculatedValuesSchema,
    effective_values: effectiveValuesSchema,
    daily_card: dailyCardSchema.nullable().default(null),
  })
  .transform(
    (v): CycleView => ({
      date: v.date,
      cycleDay: v.cycle_day,
      phase: v.phase,
      subphase: v.subphase,
      mainPhase: v.main_phase as MainPhase | null,
      fertilityLevel: v.fertility_level as CycleFertilityLevel | null,
      dataStatus: v.data_status as DataStatus | null,
      daysToOvulation: v.days_to_ovulation,
      daysToPeriod: v.days_to_period,
      daysLate: v.days_late,
      anchors: v.anchors,
      metrics: v.metrics,
      confidence: v.confidence as ConfidenceLevel | null,
      confidenceReasons: v.confidence_reasons,
      dataQuality: v.data_quality as DataQualityLevel | null,
      dataQualityDetails: v.data_quality_details,
      resolutionSource: v.resolution_source as ResolutionSource | null,
      isPredicted: v.is_predicted,
      requiresUserInput: v.requires_user_input,
      warnings: v.warnings,
      forecast: v.predictions,
      profileValues: v.profile_values,
      calculatedValues: v.calculated_values,
      effectiveValues: v.effective_values,
      dailyCard: v.daily_card,
    }),
  );

/** `{ calculation, cycle_view, calculation_status, is_recalculating }` — today / by-date. */
export const cycleCalculationEnvelopeSchema = z
  .object({
    calculation: nullableCalculationSchema.default(null),
    cycle_view: cycleViewSchema.nullable().default(null),
    calculation_status: z.string().default('completed'),
    is_recalculating: z.boolean().default(false),
  })
  .transform((e) => ({
    calculation: e.calculation,
    cycleView: e.cycle_view,
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
    // Days before `last_period_start` (or any incomplete day) come back as the
    // same empty-calculation sentinel — drop those rather than throwing.
    calculations: z.array(nullableCalculationSchema).default([]),
    calculation_status: z.string().default('completed'),
    is_recalculating: z.boolean().default(false),
    month_summary: monthSummarySchema.optional(),
  })
  .transform((e) => ({
    calculations: e.calculations.filter((c): c is CycleCalculation => c !== null),
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
