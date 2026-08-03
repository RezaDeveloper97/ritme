import type { CategoryDef, FieldControl } from './types';

// Shared control presets so the config below reads as data, not boilerplate.
const pain: FieldControl = { kind: 'degree', enumKey: 'pain_intensity' };
const bool: FieldControl = { kind: 'bool' };

/**
 * The daily-log form, grouped into the categories the Figma "Add Log" sheets
 * step through. Each category becomes a card on the Log screen and a bottom
 * sheet when tapped. This is structural config (which field uses which control /
 * enum) — the option *values* still come from `/health-logs/enums` at runtime,
 * and every visible label comes from i18n. No user-facing text lives here.
 */
export const LOG_CATEGORIES: CategoryDef[] = [
  {
    key: 'bleeding',
    fields: [
      { key: 'bleeding_intensity', control: { kind: 'chips', enumKey: 'bleeding_intensity' } },
      { key: 'blood_color', control: { kind: 'chips', enumKey: 'blood_color' } },
      { key: 'bleeding_smell', control: { kind: 'chips', enumKey: 'bleeding_smell' } },
      { key: 'clots_amount', control: { kind: 'chips', enumKey: 'clots_amount' } },
      { key: 'spotting', control: bool },
    ],
  },
  {
    key: 'pain',
    fields: [
      { key: 'stomach_ache_intensity', control: pain },
      { key: 'pelvic_pain_intensity', control: pain },
      { key: 'ovarian_pain_intensity', control: pain },
      { key: 'back_pain_intensity', control: pain },
      { key: 'headache_intensity', control: pain },
      { key: 'breast_pain_intensity', control: pain },
      { key: 'nausea_intensity', control: pain },
      { key: 'bloating_intensity', control: pain },
      { key: 'diarrhea', control: bool },
      { key: 'constipation', control: bool },
    ],
  },
  {
    // Appetite and how the body's energy felt — "اشتها و انرژی".
    key: 'digestion',
    fields: [
      { key: 'appetite_change', control: { kind: 'chips', enumKey: 'appetite_change' } },
      { key: 'food_craving', control: bool },
      { key: 'fatigue', control: bool },
      { key: 'dizziness', control: bool },
      { key: 'hot_flashes', control: bool },
      { key: 'chills', control: bool },
      { key: 'swelling', control: bool },
    ],
  },
  {
    key: 'mood',
    fields: [{ key: 'moods', control: { kind: 'multi', enumKey: 'moods' } }],
  },
  {
    key: 'sleep',
    fields: [
      { key: 'sleep_duration', control: { kind: 'chips', enumKey: 'sleep_duration' } },
      { key: 'sleep_quality', control: { kind: 'chips', enumKey: 'sleep_quality' } },
    ],
  },
  {
    key: 'exercise',
    fields: [
      // A workout day is often more than one activity, so this is multi-select.
      { key: 'exercise_type', control: { kind: 'multi', enumKey: 'exercise_type' } },
      // Minutes only, picked from a wheel — the same gesture as weight/BBT.
      {
        key: 'exercise_duration',
        control: { kind: 'measure', unit: 'minutes', min: 5, max: 300, step: 5, default: 30 },
      },
      { key: 'exercise_intensity', control: { kind: 'degree', enumKey: 'exercise_intensity' } },
    ],
  },
  {
    // Skin and hair — "پوست و مو".
    key: 'body',
    fields: [
      { key: 'acne', control: bool },
      { key: 'oily_skin', control: bool },
      { key: 'hair_loss', control: bool },
    ],
  },
  {
    key: 'discharge',
    fields: [
      { key: 'discharge_texture', control: { kind: 'chips', enumKey: 'discharge_texture' } },
      { key: 'discharge_amount', control: { kind: 'chips', enumKey: 'discharge_amount' } },
      { key: 'discharge_color', control: { kind: 'chips', enumKey: 'discharge_color' } },
      { key: 'discharge_smell', control: { kind: 'chips', enumKey: 'discharge_smell' } },
    ],
  },
  {
    key: 'intimate',
    fields: [
      { key: 'frequent_urination', control: bool },
      { key: 'urination_burning_intensity', control: pain },
      { key: 'urination_change', control: { kind: 'chips', enumKey: 'urination_change' } },
      { key: 'vaginal_dryness', control: bool },
      { key: 'vaginal_burning_intensity', control: pain },
      { key: 'vaginal_itching_intensity', control: pain },
      { key: 'vaginal_smell_change', control: bool },
    ],
  },
  {
    key: 'sexual',
    fields: [
      { key: 'sexual_desire', control: { kind: 'chips', enumKey: 'sexual_desire' } },
      { key: 'intercourse_type', control: { kind: 'chips', enumKey: 'intercourse_type' } },
      { key: 'sexual_activities', control: { kind: 'multi', enumKey: 'sexual_activities' } },
    ],
  },
  {
    key: 'weight',
    fields: [
      {
        key: 'weight',
        // The wheel is always live now, so it must open on a plausible number
        // rather than the middle of the range (90 kg / 38.5 °C would be a
        // misleading starting point the user might submit unchanged).
        control: { kind: 'measure', unit: 'kg', min: 30, max: 150, step: 0.5, default: 60, alwaysOn: true },
      },
    ],
  },
  {
    key: 'temperature',
    fields: [
      {
        key: 'basal_body_temperature',
        control: {
          kind: 'measure',
          unit: 'celsius',
          min: 35,
          max: 42,
          step: 0.1,
          default: 36.5,
          alwaysOn: true,
        },
      },
    ],
  },
  {
    key: 'notes',
    fields: [{ key: 'notes', control: { kind: 'note' } }],
  },
];
