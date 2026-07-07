import { z } from 'zod';

import type {
  AlertSummary,
  EstimatedDueDate,
  GestationalAge,
  PregnancyAlert,
  PregnancyEnums,
  PregnancyProfile,
  PregnancyStatus,
  SymptomEnums,
  WeeklyContent,
  WeeklyEnums,
} from '../model/types';

/**
 * Boundary validation for the `pregnancy` endpoints (CLAUDE.md §10). The backend
 * returns raw Eloquent models (snake_case, decimals-as-strings), so every read
 * is parsed here and mapped to the camelCase domain types. Nothing logs the
 * payload — it is sensitive health data (§11).
 */

// ── Small reusable pieces ──────────────────────────────────────
const bilingual = z.object({ en: z.string().default(''), fa: z.string().default('') });
/** Enum endpoints return `[{ value, label }]`; we keep only the values and label
 *  everything through our own i18n (`pregnancy.enums.*`). */
const enumValues = z
  .array(z.object({ value: z.union([z.string(), z.number()]).transform(String) }))
  .default([])
  .transform((rows) => rows.map((r) => r.value));

const gestationalAgeSchema = z
  .object({
    weeks: z.number().nullable().default(null),
    days: z.number().nullable().default(null),
    total_days: z.number().nullable().default(null),
    trimester: z.number().nullable().default(null),
    confidence_level: z.string().nullable().default(null),
    confidence_label: z.string().nullable().default(null),
    uncertainty_days: z.number().nullable().default(null),
    formatted: bilingual.nullable().default(null),
  })
  .transform(
    (g): GestationalAge => ({
      weeks: g.weeks,
      days: g.days,
      totalDays: g.total_days,
      trimester: g.trimester,
      confidenceLevel: g.confidence_level,
      confidenceLabel: g.confidence_label,
      uncertaintyDays: g.uncertainty_days,
      formatted: g.formatted,
    }),
  );

const estimatedDueDateSchema = z
  .object({
    date: z.string(),
    days_remaining: z.number().default(0),
    weeks_remaining: z.number().default(0),
    formatted: bilingual.default({ en: '', fa: '' }),
  })
  .transform(
    (d): EstimatedDueDate => ({
      date: d.date,
      daysRemaining: d.days_remaining,
      weeksRemaining: d.weeks_remaining,
      formatted: d.formatted,
    }),
  );

// ── Status ─────────────────────────────────────────────────────
export const pregnancyStatusSchema = z
  .object({
    is_active: z.boolean().default(false),
    gestational_age: gestationalAgeSchema.nullish(),
    estimated_due_date: estimatedDueDateSchema.nullish(),
    current_week: z.number().nullable().default(null),
    trimester: z.number().nullable().default(null),
    age_source: z.string().nullable().default(null),
    confidence_level: z.string().nullable().default(null),
    is_high_risk: z.boolean().default(false),
    fetal_movement_tracking_active: z.boolean().default(false),
    fetal_movement_required: z.boolean().default(false),
    flags: z.record(z.union([bilingual, z.string()])).default({}),
  })
  .transform(
    (s): PregnancyStatus => ({
      isActive: s.is_active,
      gestationalAge: s.gestational_age ?? null,
      estimatedDueDate: s.estimated_due_date ?? null,
      currentWeek: s.current_week,
      trimester: s.trimester,
      ageSource: s.age_source,
      confidenceLevel: s.confidence_level,
      isHighRisk: s.is_high_risk,
      fetalMovementTrackingActive: s.fetal_movement_tracking_active,
      fetalMovementRequired: s.fetal_movement_required,
      flags: s.flags,
    }),
  );

// ── Profile ────────────────────────────────────────────────────
const bool = z.boolean().nullish();
export const pregnancyProfileSchema = z
  .object({
    pregnancy_mode: z.boolean().default(true),
    onboarding_completed: z.boolean().default(false),
    is_locked: z.boolean().default(false),
    age_source: z.string().nullish(),
    confidence_level: z.string().nullish(),
    lmp_date: z.string().nullish(),
    ultrasound_date: z.string().nullish(),
    ultrasound_weeks: z.number().nullish(),
    ultrasound_days: z.number().nullish(),
    manual_weeks: z.number().nullish(),
    manual_days: z.number().nullish(),
    estimated_due_date: z.string().nullish(),
    uncertainty_days: z.number().nullish(),
    has_miscarriage_history: bool,
    has_high_risk_history: bool,
    pre_existing_conditions: z.array(z.string()).nullish(),
    blood_type: z.string().nullish(),
    rh_factor: z.string().nullish(),
    rh_negative_care_flag: z.boolean().default(false),
    fetal_movement_felt: z.boolean().default(false),
    first_fetal_movement_date: z.string().nullish(),
  })
  .transform(
    (p): PregnancyProfile => ({
      pregnancyMode: p.pregnancy_mode,
      onboardingCompleted: p.onboarding_completed,
      isLocked: p.is_locked,
      ageSource: p.age_source ?? null,
      confidenceLevel: p.confidence_level ?? null,
      lmpDate: p.lmp_date ?? null,
      ultrasoundDate: p.ultrasound_date ?? null,
      ultrasoundWeeks: p.ultrasound_weeks ?? null,
      ultrasoundDays: p.ultrasound_days ?? null,
      manualWeeks: p.manual_weeks ?? null,
      manualDays: p.manual_days ?? null,
      estimatedDueDate: p.estimated_due_date ?? null,
      uncertaintyDays: p.uncertainty_days ?? null,
      hasMiscarriageHistory: p.has_miscarriage_history ?? null,
      hasHighRiskHistory: p.has_high_risk_history ?? null,
      preExistingConditions: p.pre_existing_conditions ?? [],
      bloodType: p.blood_type ?? null,
      rhFactor: p.rh_factor ?? null,
      rhNegativeCareFlag: p.rh_negative_care_flag,
      fetalMovementFelt: p.fetal_movement_felt,
      firstFetalMovementDate: p.first_fetal_movement_date ?? null,
    }),
  );

/** GET /pregnancy/profile → `{ profile, status }`. */
export const pregnancyProfileEnvelopeSchema = z.object({
  profile: pregnancyProfileSchema,
  status: pregnancyStatusSchema,
});

// ── Enums ──────────────────────────────────────────────────────
export const pregnancyEnumsSchema = z
  .object({
    age_sources: enumValues,
    confidence_levels: enumValues,
    blood_types: enumValues,
    rh_factors: enumValues,
    pre_existing_conditions: enumValues,
    alert_levels: enumValues,
  })
  .transform(
    (e): PregnancyEnums => ({
      ageSources: e.age_sources,
      confidenceLevels: e.confidence_levels,
      bloodTypes: e.blood_types,
      rhFactors: e.rh_factors,
      preExistingConditions: e.pre_existing_conditions,
      alertLevels: e.alert_levels,
    }),
  );

export const symptomEnumsSchema = z
  .object({ severity: enumValues })
  .transform((e): SymptomEnums => ({ severity: e.severity }));

export const weeklyEnumsSchema = z
  .object({
    swelling_locations: enumValues,
    overall_mood: enumValues,
    severity: enumValues,
    fetal_movement_status: enumValues,
  })
  .transform(
    (e): WeeklyEnums => ({
      swellingLocations: e.swelling_locations,
      overallMood: e.overall_mood,
      severity: e.severity,
      fetalMovementStatus: e.fetal_movement_status,
    }),
  );

// ── Weekly content ─────────────────────────────────────────────
/** A module is a localized string; the endpoint resolves the locale for us, but
 *  we also tolerate a raw `{en,fa}` object just in case. */
const contentField = z
  .union([z.string(), bilingual, z.null()])
  .transform((v) => (v == null ? null : typeof v === 'string' ? v : v.fa || v.en || null));

const faqItem = z.object({
  question: z.string().default(''),
  answer: z.string().default(''),
});

export const weeklyContentSchema = z
  .object({
    week_number: z.number(),
    fetal_development: contentField.default(null),
    mother_body_changes: contentField.default(null),
    dos_and_donts: contentField.default(null),
    care_plan: contentField.default(null),
    body_adaptation: contentField.default(null),
    emotional_status: contentField.default(null),
    key_nutrition: contentField.default(null),
    physical_activity: contentField.default(null),
    tests_and_checkups: contentField.default(null),
    faq: z.array(faqItem).nullish(),
  })
  .transform(
    (c): WeeklyContent => ({
      weekNumber: c.week_number,
      fetalDevelopment: c.fetal_development,
      motherBodyChanges: c.mother_body_changes,
      dosAndDonts: c.dos_and_donts,
      carePlan: c.care_plan,
      bodyAdaptation: c.body_adaptation,
      emotionalStatus: c.emotional_status,
      keyNutrition: c.key_nutrition,
      physicalActivity: c.physical_activity,
      testsAndCheckups: c.tests_and_checkups,
      faq: c.faq ?? [],
    }),
  );

/** GET /pregnancy/content/{week} → `{ week, content }`; we unwrap to content. */
export const weeklyContentEnvelopeSchema = z
  .object({ week: z.number(), content: weeklyContentSchema })
  .transform((e) => e.content);

// ── Alerts ─────────────────────────────────────────────────────
export const pregnancyAlertSchema = z
  .object({
    id: z.number(),
    alert_level: z.enum(['info', 'warning', 'emergency']).catch('info'),
    alert_type: z.string().default(''),
    title: z.string().default(''),
    message: z.string().default(''),
    pregnancy_week: z.number().nullish(),
    recommended_actions: z.array(z.string()).nullish(),
    is_read: z.boolean().default(false),
    is_dismissed: z.boolean().default(false),
    created_at: z.string().nullish(),
  })
  .transform(
    (a): PregnancyAlert => ({
      id: a.id,
      alertLevel: a.alert_level,
      alertType: a.alert_type,
      title: a.title,
      message: a.message,
      pregnancyWeek: a.pregnancy_week ?? null,
      recommendedActions: a.recommended_actions ?? [],
      isRead: a.is_read,
      isDismissed: a.is_dismissed,
      createdAt: a.created_at ?? null,
    }),
  );

const alertCounts = z.object({
  total: z.number().default(0),
  unread: z.number().default(0),
  emergency: z.number().default(0),
  warning: z.number().default(0),
  info: z.number().default(0),
});

export const alertsEnvelopeSchema = z.object({
  alerts: z.array(pregnancyAlertSchema).default([]),
  counts: alertCounts,
});

export const alertSummarySchema = z
  .object({
    has_emergency: z.boolean().default(false),
    has_unread: z.boolean().default(false),
    counts: alertCounts,
    latest_emergency: pregnancyAlertSchema.nullish(),
  })
  .transform(
    (s): AlertSummary => ({
      hasEmergency: s.has_emergency,
      hasUnread: s.has_unread,
      counts: s.counts,
      latestEmergency: s.latest_emergency ?? null,
    }),
  );
