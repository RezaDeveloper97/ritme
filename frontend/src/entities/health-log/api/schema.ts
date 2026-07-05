import { z } from 'zod';

import type { HealthLogEnums, HealthLogInput } from '../model/types';

/**
 * Boundary validation for the `health-logs` endpoints (CLAUDE.md §10). We keep
 * the API's snake_case field names (§8.1) so the daily log is one object end to
 * end. Privacy (§11): nothing here logs the payload — it's health data.
 */

const enumList = z.array(z.string()).default([]);

/** GET /health-logs/enums → the option lists that drive the form. */
export const healthLogEnumsSchema = z
  .object({
    bleeding_intensity: enumList,
    blood_color: enumList,
    bleeding_smell: enumList,
    pain_intensity: enumList,
    appetite_change: enumList,
    urination_change: enumList,
    moods: enumList,
    sleep_duration: enumList,
    sleep_quality: enumList,
    sexual_activities: enumList,
    discharge_texture: enumList,
    discharge_amount: enumList,
    discharge_smell: enumList,
  })
  .transform((e): HealthLogEnums => e);

const enumValue = z.string().nullish();
const flag = z.boolean().nullish();
const stringList = z.array(z.string()).nullish();
const measure = z.coerce.number().nullish();

/**
 * GET /health-logs/{date} → `DailyHealthLog`. We validate only the fields the
 * form edits and drop the rest (`id`, `user_id`, timestamps). Nulls are
 * stripped so the result is a clean {@link HealthLogInput} patch for prefilling.
 */
export const dailyHealthLogSchema = z
  .object({
    log_date: z.string(),
    bleeding_intensity: enumValue,
    blood_color: enumValue,
    bleeding_smell: enumValue,
    has_clots: flag,
    spotting: flag,
    headache_intensity: enumValue,
    stomach_ache_intensity: enumValue,
    pelvic_pain_intensity: enumValue,
    breast_pain_intensity: enumValue,
    back_pain_intensity: enumValue,
    ovarian_pain_intensity: enumValue,
    nausea_intensity: enumValue,
    bloating_intensity: enumValue,
    breast_sensitivity_intensity: enumValue,
    urination_burning_intensity: enumValue,
    appetite_change: enumValue,
    diarrhea: flag,
    constipation: flag,
    food_craving: flag,
    moods: stringList,
    sleep_duration: enumValue,
    sleep_quality: enumValue,
    acne: flag,
    oily_skin: flag,
    hair_loss: flag,
    swelling: flag,
    fatigue: flag,
    dizziness: flag,
    hot_flashes: flag,
    chills: flag,
    discharge_texture: enumValue,
    discharge_amount: enumValue,
    discharge_smell: enumValue,
    discharge_itching: flag,
    discharge_burning: flag,
    frequent_urination: flag,
    vaginal_dryness: flag,
    vaginal_burning: flag,
    vaginal_itching: flag,
    vaginal_smell_change: flag,
    urination_change: enumValue,
    sexual_activities: stringList,
    weight: measure,
    basal_body_temperature: measure,
    notes: z.string().nullish(),
  })
  .transform((raw): HealthLogInput => {
    // Drop null/undefined so a prefilled draft only carries recorded values.
    const out: HealthLogInput = { log_date: raw.log_date };
    for (const [key, value] of Object.entries(raw)) {
      if (key === 'log_date' || value === null || value === undefined) continue;
      (out as unknown as Record<string, unknown>)[key] = value;
    }
    return out;
  });
