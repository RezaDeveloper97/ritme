import {
  DAY_SYMPTOMS,
  type DaySymptoms,
  type PregnancyDayInput,
  WATER_MAX,
  WATER_MIN,
} from '@/entities/pregnancy';

/**
 * Log screen input → the `PUT /pregnancy/v2/days/{date}` body (T-M7-03).
 *
 * - `undefined` fields are omitted (leave as is), explicit `null` is sent
 *   (clear) — the day-log contract;
 * - symptoms keep canonical order, unknown keys are dropped; `{}` clears them;
 * - water is clamped to the stepper's 0–15; weight is rounded to 0.1 kg and a
 *   non-positive / non-finite weight is sent as `null`;
 * - a blank visit note becomes `null`.
 *
 * The same body is what an offline outbox would replay (T-M7-12), so it must
 * be a pure function of the input — no timestamps, no ids.
 */
export function toPregnancyDayBody(input: PregnancyDayInput): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.mood !== undefined) body.mood = input.mood;
  if (input.symptoms !== undefined) body.symptoms = canonicalSymptoms(input.symptoms);
  if (input.waterGlasses !== undefined) {
    body.water_glasses =
      input.waterGlasses === null || !Number.isFinite(input.waterGlasses)
        ? null
        : Math.min(WATER_MAX, Math.max(WATER_MIN, Math.round(input.waterGlasses)));
  }
  if (input.weight !== undefined) {
    body.weight =
      input.weight === null || !Number.isFinite(input.weight) || input.weight <= 0
        ? null
        : Math.round(input.weight * 10) / 10;
  }
  if (input.visitNote !== undefined) body.visit_note = input.visitNote?.trim() || null;
  return body;
}

function canonicalSymptoms(symptoms: DaySymptoms): DaySymptoms {
  const out: DaySymptoms = {};
  for (const key of DAY_SYMPTOMS) {
    const severity = symptoms[key];
    if (severity) out[key] = severity;
  }
  return out;
}
