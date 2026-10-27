import { z } from 'zod';

import type {
  CalculatedValues,
  CardAction,
  CycleDailyTip,
  ConfidenceLevel,
  CycleAnchors,
  CycleCalculation,
  CycleClinicalFlags,
  CycleFutureCycle,
  CycleMonthDay,
  CycleLimits,
  CycleRegularity,
  CycleTodayUi,
  OvulationSource,
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
/**
 * One phase/symptom-driven tip from the engine. The backend localizes `text` and
 * `title` to the request locale; `type` is a stable category code the UI maps to
 * an icon. `title` is optional — the recommendations are admin-managed, and an
 * older backend sent none. Health copy is display-only and never logged (§11).
 */
const dailyTipSchema = z
  .object({
    type: z.string().default('general'),
    // `nullish`, not `optional`: a parse failure here would drop the whole tip
    // (the list is `safeParse`d per item), so an explicit null must cost the
    // heading, never the advice. Blank titles collapse to undefined so the
    // "present means non-empty" contract holds for callers.
    title: z.string().nullish(),
    text: z.string(),
  })
  .transform((t): CycleDailyTip => ({
    type: t.type,
    title: t.title?.trim() || undefined,
    text: t.text,
  }));

export const cycleCalculationSchema = z
  .object({
    // Gregorian `YYYY-MM-DD` the backend stamps on every calculation. The
    // calendar keys each day cell by this so a filtered/sparse month array still
    // maps to the right dates (§7 — converted to the locale's calendar only for display).
    calculation_date: z.string(),
    cycle_day: z.number(),
    phase: z.string(),
    subphase: z.string().nullable().default(null),
    // Null under hormonal contraception / suppressed fertility (v6) — never throw on it.
    estimated_ovulation_day: z.number().nullable().default(null),
    cycle_length_used: z.number(),
    is_fertile_window: z.boolean().default(false),
    is_pms_window: z.boolean().default(false),
    is_period_tomorrow: z.boolean().default(false),
    final_probability: z.number().default(0),
    cycle_variability: z.string().nullable().default(null),
    // Tolerate the pre-localization shape (`{type, en, fa}`) an older backend
    // still sends by dropping entries without a `text` rather than throwing.
    daily_tips: z
      .array(z.unknown())
      .default([])
      .transform((tips) =>
        tips.map((t) => dailyTipSchema.safeParse(t)).flatMap((r) => (r.success ? [r.data] : [])),
      ),
  })
  .transform((c): CycleCalculation => ({
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
    dailyTips: c.daily_tips,
  }));

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
      .enum(['very_low', 'low', 'medium', 'high', 'very_high', 'peak', 'unknown'])
      .catch('unknown'),
    fertility_label: z.string().default(''),
    badges: z.array(z.string()).default([]),
    primary_action: cardActionSchema.nullable().default(null),
    secondary_actions: z.array(cardActionSchema).default([]),
  })
  .transform((c): DailyCard => ({
    title: c.title,
    subtitle: c.subtitle,
    dataStatus: c.data_status as DataStatus,
    fertilityLevel: c.fertility_level,
    fertilityLabel: c.fertility_label,
    badges: c.badges,
    primaryAction: c.primary_action,
    secondaryActions: c.secondary_actions,
  }));

const forecastSchema = z
  .object({
    // Every date may be null in v6 (hormonal contraception, unresolved anchors).
    next_period_start: z.string().nullable().default(null),
    next_period_end: z.string().nullable().default(null),
    estimated_ovulation_date: z.string().nullable().default(null),
    fertile_window_start: z.string().nullable().default(null),
    fertile_window_end: z.string().nullable().default(null),
    source: z.string().default('profile'),
    confidence: z.string().default('low'),
    confidence_reasons: z.array(z.string()).default([]),
  })
  .transform((p): CycleForecast => ({
    nextPeriodStart: p.next_period_start,
    nextPeriodEnd: p.next_period_end,
    estimatedOvulationDate: p.estimated_ovulation_date,
    fertileWindowStart: p.fertile_window_start,
    fertileWindowEnd: p.fertile_window_end,
    source: p.source,
    confidence: p.confidence as ConfidenceLevel,
    confidenceReasons: p.confidence_reasons,
  }));

const OVULATION_SOURCES = ['bbt_confirmed', 'lh_positive', 'calendar', 'unknown'] as const;
const CYCLE_REGULARITIES = ['regular', 'irregular', 'insufficient_data'] as const;
const MAIN_PHASES = [
  'menstrual',
  'follicular',
  'fertile',
  'luteal',
  'period_expected',
  'unknown',
] as const;
const CYCLE_FERTILITY_LEVELS = ['none', 'low', 'medium', 'high', 'peak', 'unknown'] as const;
const CONFIDENCE_LEVELS = ['high', 'medium', 'low', 'unknown'] as const;
const RESOLUTION_SOURCES = [
  'user_logged',
  'user_logged_with_assumed_end',
  'prediction',
  'onboarding_based',
  'default_based',
  'unknown',
] as const;

/**
 * An enum-ish string field that never throws: null/absent stays `null`, a known
 * value passes, anything a newer backend invents degrades to `'unknown'`.
 */
const lenientEnum = <T extends readonly string[]>(values: T) =>
  z
    .string()
    .nullish()
    .transform((v): T[number] | null =>
      v == null ? null : (values as readonly string[]).includes(v) ? (v as T[number]) : 'unknown',
    );

/** A value-layer object that may be absent or null in v6 — default to an empty layer. */
const orEmptyObject = <S extends z.ZodTypeAny>(schema: S) =>
  z.preprocess((v) => (v == null || typeof v !== 'object' ? {} : v), schema);

const valueLayerFields = {
  cycle_length: z.number().nullable().default(null),
  period_duration: z.number().nullable().default(null),
};

const profileValuesSchema = z.object(valueLayerFields).transform((v): CycleValueLayer => ({
  cycleLength: v.cycle_length,
  periodDuration: v.period_duration,
}));

const calculatedValuesSchema = z
  .object({
    ...valueLayerFields,
    based_on_cycles: z.number().nullable().default(null),
  })
  .transform((v): CalculatedValues => ({
    cycleLength: v.cycle_length,
    periodDuration: v.period_duration,
    basedOnCycles: v.based_on_cycles,
  }));

const effectiveValuesSchema = z
  .object({
    ...valueLayerFields,
    source: z.string().default('default'),
    period_duration_source: z.string().default('default'),
  })
  .transform((v): EffectiveValues => ({
    cycleLength: v.cycle_length,
    periodDuration: v.period_duration,
    source: v.source,
    periodDurationSource: v.period_duration_source,
  }));

const dataQualityDetailsSchema = z
  .object({
    confidence: z.string().default('low'),
    confidence_reasons: z.array(z.string()).default([]),
    regularity_status: z.string().default('not_enough_data'),
    is_irregular_possible: z.boolean().default(false),
    missing_period_end: z.boolean().default(false),
  })
  .transform((q): CycleDataQuality => ({
    confidence: q.confidence as ConfidenceLevel,
    confidenceReasons: q.confidence_reasons,
    regularityStatus: q.regularity_status,
    isIrregularPossible: q.is_irregular_possible,
    missingPeriodEnd: q.missing_period_end,
  }));

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
    ovulation_source: z.enum(OVULATION_SOURCES).catch('unknown'),
  })
  .transform((a): CycleAnchors => ({
    currentPeriodStart: a.current_period_start,
    currentPeriodStartSource: a.current_period_start_source,
    currentPeriodEnd: a.current_period_end,
    currentPeriodEndSource: a.current_period_end_source,
    currentPeriodEndIsConfirmed: a.current_period_end_is_confirmed,
    predictedNextPeriodStart: a.predicted_next_period_start,
    estimatedOvulationDate: a.estimated_ovulation_date,
    ovulationSource: a.ovulation_source as OvulationSource,
  }));

/** The v1.1 effective-metrics block (task.md §9, §27, §35). */
const engineMetricsSchema = z
  .object({
    effective_cycle_length: z.number().nullable().default(null),
    effective_period_length: z.number().nullable().default(null),
    cycle_variability: z.number().nullable().default(null),
    cycle_regularity: z.enum(CYCLE_REGULARITIES).nullable().catch(null),
  })
  .transform((m): CycleEngineMetrics => ({
    effectiveCycleLength: m.effective_cycle_length,
    effectivePeriodLength: m.effective_period_length,
    cycleVariability: m.cycle_variability,
    cycleRegularity: m.cycle_regularity as CycleRegularity | null,
  }));

/** One predicted future cycle (`k` = 2 is the cycle after next). Dates are Gregorian. */
const futureCycleSchema = z
  .object({
    k: z.number(),
    period_start: z.string().nullable().default(null),
    ovulation: z.string().nullable().default(null),
  })
  .transform((f): CycleFutureCycle => ({
    k: f.k,
    periodStart: f.period_start,
    ovulation: f.ovulation,
  }));

/** FIGO-style clinical read-out (§2.27). Unknown labels pass through as strings. */
const clinicalFlagsSchema = z
  .object({
    frequency: z.string().nullable().catch(null),
    duration: z.string().nullable().catch(null),
    absent: z.boolean().catch(false),
  })
  .transform((c): CycleClinicalFlags => ({
    frequency: c.frequency,
    duration: c.duration,
    absent: c.absent,
  }));

const dateRangeSchema = z
  .object({
    start: z.string().nullable().catch(null),
    end: z.string().nullable().catch(null),
  })
  .nullable()
  .catch(null);

/** The engine's pure Today mapping (E-17). Every inner field degrades, none throws. */
const todayUiSchema = z
  .object({
    ring_action: z.string().nullable().catch(null),
    banner: z.string().nullable().catch(null),
    show_pregnancy_test_button: z.boolean().catch(false),
    prediction_card: z
      .object({
        period: dateRangeSchema,
        pms: dateRangeSchema,
        fertile: dateRangeSchema,
        ovulation: z
          .object({
            date: z.string().nullable().catch(null),
            source: z.enum(OVULATION_SOURCES).catch('unknown'),
          })
          .nullable()
          .catch(null),
        cycle_length: z.number().nullable().catch(null),
        cycle_regularity: z.enum(CYCLE_REGULARITIES).nullable().catch(null),
      })
      .nullable()
      .catch(null),
  })
  .transform((t): CycleTodayUi => {
    const card = t.prediction_card;
    return {
      ringAction: t.ring_action,
      banner: t.banner,
      showPregnancyTestButton: t.show_pregnancy_test_button,
      predictionCard: card
        ? {
            period: card.period,
            pms: card.pms,
            fertile: card.fertile,
            ovulation: card.ovulation
              ? {
                  date: card.ovulation.date,
                  source: card.ovulation.source as OvulationSource,
                }
              : null,
            cycleLength: card.cycle_length,
            cycleRegularity: card.cycle_regularity as CycleRegularity | null,
          }
        : null,
    };
  });

/** Output limits (E-24): clinical suppression flags a client must respect. */
const limitsSchema = z
  .object({
    fertility_suppressed: z.boolean().catch(false),
    suppression_reason: z.string().nullable().catch(null),
    engine_suspended: z.boolean().catch(false),
    prompt_pregnancy_mode: z.boolean().catch(false),
  })
  .transform((l): CycleLimits => ({
    fertilitySuppressed: l.fertility_suppressed,
    suppressionReason: l.suppression_reason,
    engineSuspended: l.engine_suspended,
    promptPregnancyMode: l.prompt_pregnancy_mode,
  }));

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
    main_phase: lenientEnum(MAIN_PHASES),
    fertility_level: lenientEnum(CYCLE_FERTILITY_LEVELS),
    data_status: z.string().nullable().default(null),
    days_to_ovulation: z.number().nullable().default(null),
    days_to_period: z.number().nullable().default(null),
    days_late: z.number().nullable().default(null),
    anchors: anchorsSchema.nullable().catch(null),
    metrics: engineMetricsSchema.nullable().catch(null),
    confidence: lenientEnum(CONFIDENCE_LEVELS),
    confidence_reasons: z.array(z.string()).default([]),
    data_quality: z.string().nullable().default(null),
    data_quality_details: dataQualityDetailsSchema.nullable().catch(null),
    resolution_source: lenientEnum(RESOLUTION_SOURCES),
    is_predicted: z.boolean().default(false),
    requires_user_input: z.boolean().default(false),
    warnings: z.array(z.string()).default([]),
    predictions: forecastSchema.nullable().catch(null),
    profile_values: orEmptyObject(profileValuesSchema),
    calculated_values: orEmptyObject(calculatedValuesSchema),
    effective_values: orEmptyObject(effectiveValuesSchema),
    daily_card: dailyCardSchema.nullable().catch(null),
    // --- v6 additions: all optional, none may fail the parse ---
    future_cycles: z
      .array(z.unknown())
      .catch([])
      .transform((items) =>
        items.flatMap((i) => {
          const r = futureCycleSchema.safeParse(i);
          return r.success ? [r.data] : [];
        }),
      ),
    clinical_flags: clinicalFlagsSchema.nullable().catch(null),
    suggest_pregnancy_test: z.boolean().catch(false).default(false),
    today_ui: todayUiSchema.nullable().catch(null),
    limits: limitsSchema.nullable().catch(null),
  })
  .transform((v): CycleView => ({
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
    futureCycles: v.future_cycles,
    clinicalFlags: v.clinical_flags,
    suggestPregnancyTest: v.suggest_pregnancy_test,
    todayUi: v.today_ui,
    limits: v.limits,
  }));

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
  .transform((s): MonthSummary => ({
    fertileDays: s.fertile_days,
    periodDays: s.period_days,
    pmsDays: s.pms_days,
  }));

/** One v6 month day (D-121): the paint fields, parsed leniently — a bad field degrades to null. */
const monthDaySchema = z
  .object({
    calculation_date: z.string(),
    main_phase: lenientEnum(MAIN_PHASES),
    fertility_level: lenientEnum(CYCLE_FERTILITY_LEVELS),
    cycle_day: z.number().nullable().catch(null),
    estimated_ovulation_day: z.number().nullable().catch(null),
    phase: z.string().nullable().catch(null),
    is_predicted: z.boolean().catch(true),
    is_period: z.boolean().catch(false),
    period_kind: z.enum(['logged', 'assumed', 'predicted']).nullable().catch(null),
    days_to_period: z.number().nullable().catch(null),
    days_late: z.number().nullable().catch(null),
  })
  .transform((d): CycleMonthDay => ({
    date: d.calculation_date,
    mainPhase: d.main_phase as MainPhase | null,
    fertilityLevel: d.fertility_level as CycleFertilityLevel | null,
    cycleDay: d.cycle_day,
    estimatedOvulationDay: d.estimated_ovulation_day,
    phase: d.phase,
    isPredicted: d.is_predicted,
    isPeriod: d.is_period,
    periodKind: d.period_kind,
    daysToPeriod: d.days_to_period,
    daysLate: d.days_late,
  }));

/** Whether a raw month day carries the v6 keys (Laravel / v1 days never do). */
const isV6MonthDay = (v: unknown): boolean =>
  !!v && typeof v === 'object' && ('period_kind' in v || 'main_phase' in v);

/** One predicted future cycle of the month payload (k ≤ 6). Dates are Gregorian. */
const monthFutureCycleSchema = z
  .object({
    k: z.number(),
    period_start: z.string().nullable().catch(null),
    period_end: z.string().nullable().catch(null),
    fertile_start: z.string().nullable().catch(null),
    fertile_end: z.string().nullable().catch(null),
    ovulation: z.string().nullable().catch(null),
  })
  .transform((f): CycleFutureCycle => ({
    k: f.k,
    periodStart: f.period_start,
    periodEnd: f.period_end,
    fertileStart: f.fertile_start,
    fertileEnd: f.fertile_end,
    ovulation: f.ovulation,
  }));

/** `{ calculations[], month_summary, … }` — a month of calculations. */
export const cycleMonthEnvelopeSchema = z
  .object({
    // Days before `last_period_start` (or any incomplete day) come back as the
    // same empty-calculation sentinel — drop those rather than throwing.
    // Parsed one by one: a malformed day is dropped, never the whole month.
    // v6 days the engine could not place (null cycle day) are dropped from
    // `calculations` too; they live on in `days`.
    calculations: z
      .array(z.unknown())
      .catch([])
      .transform((days) => ({
        calculations: days.flatMap((d) => {
          const r = nullableCalculationSchema.safeParse(d);
          return r.success && r.data !== null ? [r.data] : [];
        }),
        days: days.flatMap((d) => {
          if (!isV6MonthDay(d)) return [];
          const r = monthDaySchema.safeParse(d);
          return r.success ? [r.data] : [];
        }),
      })),
    calculation_status: z.string().default('completed'),
    is_recalculating: z.boolean().default(false),
    month_summary: monthSummarySchema.optional(),
    // --- v6 additions (D-121): absent on a Laravel payload ---
    future_cycles: z
      .array(z.unknown())
      .catch([])
      .transform((items) =>
        items.flatMap((i) => {
          const r = monthFutureCycleSchema.safeParse(i);
          return r.success ? [r.data] : [];
        }),
      ),
    limits: limitsSchema.nullable().catch(null),
  })
  .transform((e) => ({
    calculations: e.calculations.calculations as CycleCalculation[],
    /** v6 per-day paint data; empty on a Laravel / v1 payload. */
    days: e.calculations.days,
    calculationStatus: e.calculation_status,
    isRecalculating: e.is_recalculating,
    monthSummary: e.month_summary ?? null,
    /** v6 `future_cycles` (k ≤ 6); empty on Laravel and when nothing is predicted. */
    futureCycles: e.future_cycles,
    /** v6 `limits`; null on Laravel. */
    limits: e.limits,
  }));

/** `{ status, is_processing, version, … }` — calculation processing status. */
export const cycleStatusSchema = z.object({
  status: z.string().default('completed'),
  status_label: z.string().default(''),
  is_processing: z.boolean().default(false),
  version: z.number().default(1),
});
