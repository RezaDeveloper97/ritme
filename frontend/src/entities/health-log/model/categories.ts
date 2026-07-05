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
      { key: 'has_clots', control: bool },
      { key: 'spotting', control: bool },
    ],
  },
  {
    key: 'pain',
    fields: [
      { key: 'headache_intensity', control: pain },
      { key: 'stomach_ache_intensity', control: pain },
      { key: 'pelvic_pain_intensity', control: pain },
      { key: 'ovarian_pain_intensity', control: pain },
      { key: 'back_pain_intensity', control: pain },
      { key: 'breast_pain_intensity', control: pain },
      { key: 'breast_sensitivity_intensity', control: pain },
      { key: 'nausea_intensity', control: pain },
      { key: 'bloating_intensity', control: pain },
    ],
  },
  {
    key: 'digestion',
    fields: [
      { key: 'appetite_change', control: { kind: 'chips', enumKey: 'appetite_change' } },
      { key: 'food_craving', control: bool },
      { key: 'diarrhea', control: bool },
      { key: 'constipation', control: bool },
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
    key: 'body',
    fields: [
      { key: 'fatigue', control: bool },
      { key: 'dizziness', control: bool },
      { key: 'hot_flashes', control: bool },
      { key: 'chills', control: bool },
      { key: 'swelling', control: bool },
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
      { key: 'discharge_smell', control: { kind: 'chips', enumKey: 'discharge_smell' } },
      { key: 'discharge_itching', control: bool },
      { key: 'discharge_burning', control: bool },
    ],
  },
  {
    key: 'intimate',
    fields: [
      { key: 'urination_change', control: { kind: 'chips', enumKey: 'urination_change' } },
      { key: 'urination_burning_intensity', control: pain },
      { key: 'frequent_urination', control: bool },
      { key: 'vaginal_dryness', control: bool },
      { key: 'vaginal_burning', control: bool },
      { key: 'vaginal_itching', control: bool },
      { key: 'vaginal_smell_change', control: bool },
    ],
  },
  {
    key: 'sexual',
    fields: [{ key: 'sexual_activities', control: { kind: 'multi', enumKey: 'sexual_activities' } }],
  },
  {
    key: 'measure',
    fields: [
      { key: 'weight', control: { kind: 'measure', unit: 'kg', min: 30, max: 150, step: 0.5 } },
      {
        key: 'basal_body_temperature',
        control: { kind: 'measure', unit: 'celsius', min: 35, max: 42, step: 0.1 },
      },
    ],
  },
  {
    key: 'notes',
    fields: [{ key: 'notes', control: { kind: 'note' } }],
  },
];
