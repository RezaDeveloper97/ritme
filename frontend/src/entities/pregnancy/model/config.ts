import type { ContentModuleKey, SymptomKey } from './types';

/**
 * Static shape of the pregnancy log forms. These are UI-driving *keys* only —
 * labels come from i18n (`pregnancy.*`) and option values from the `/enums`
 * endpoints, so nothing medical is hardcoded here (§8.1).
 */

/** All 14 daily-tracked symptoms, grouped for the log sheet. */
export const SYMPTOM_GROUPS: { key: string; symptoms: SymptomKey[] }[] = [
  { key: 'common', symptoms: ['nausea', 'vomiting', 'fatigue', 'headache', 'dizziness'] },
  {
    key: 'aches',
    symptoms: ['breast_pain', 'lower_abdominal_pain', 'cramping', 'back_pain', 'pelvic_pressure'],
  },
  { key: 'warning', symptoms: ['spotting', 'bleeding', 'fluid_leakage', 'severe_sudden_pain'] },
];

/** Symptoms that can raise an alert on save — surfaced with a warning accent. */
export const CRITICAL_SYMPTOMS: SymptomKey[] = [
  'spotting',
  'bleeding',
  'fluid_leakage',
  'severe_sudden_pain',
];

export function isCriticalSymptom(key: SymptomKey): boolean {
  return CRITICAL_SYMPTOMS.includes(key);
}

/** Whether a symptom offers a severity follow-up (all of them do here). */
export const SYMPTOM_KEYS: SymptomKey[] = SYMPTOM_GROUPS.flatMap((g) => g.symptoms);

/** Mental-health items on the weekly checkup (each is a bool + severity pair). */
export const WEEKLY_MENTAL: { flag: string; severity: string }[] = [
  { flag: 'has_anxiety', severity: 'anxiety_severity' },
  { flag: 'has_mood_swings', severity: 'mood_swings_severity' },
  { flag: 'has_depression_feelings', severity: 'depression_severity' },
];

/** Long-form content modules in reading order (faq is rendered separately). */
export const CONTENT_MODULE_ORDER: ContentModuleKey[] = [
  'fetalDevelopment',
  'motherBodyChanges',
  'bodyAdaptation',
  'emotionalStatus',
  'keyNutrition',
  'physicalActivity',
  'dosAndDonts',
  'carePlan',
  'testsAndCheckups',
];

export const TOTAL_WEEKS = 40;
