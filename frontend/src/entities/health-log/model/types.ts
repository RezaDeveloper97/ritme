/**
 * Domain types for the daily health log (the "Add Log" flow — CLAUDE.md §5,
 * `health-logs` group in §8.1). Field names mirror the API's snake_case payload
 * so a single object crosses the boundary without a 48-field rename map.
 *
 * Enum-valued fields are typed as `string` on purpose: the option lists are
 * driven by the `/health-logs/enums` endpoint (§8.1 — derive from enums, never
 * hardcode), so we validate values at runtime against the fetched enums rather
 * than freezing them into brittle literal unions here.
 */

/** The keys of the `/health-logs/enums` response — one option list per key. */
export type EnumKey =
  | 'bleeding_intensity'
  | 'blood_color'
  | 'bleeding_smell'
  | 'pain_intensity'
  | 'appetite_change'
  | 'urination_change'
  | 'moods'
  | 'sleep_duration'
  | 'sleep_quality'
  | 'sexual_activities'
  | 'discharge_texture'
  | 'discharge_amount'
  | 'discharge_smell';

/** Available option values for each enum, as returned by the API. */
export type HealthLogEnums = Record<EnumKey, string[]>;

/**
 * The daily log payload. Every field except `log_date` is optional — the user
 * records only what applies to their day. This is exactly the POST /health-logs
 * body and the useful subset of the GET response (§8.1).
 */
export interface HealthLogInput {
  /** Gregorian `YYYY-MM-DD` (from `shared/lib/date` `toApiDate`). */
  log_date: string;

  // ── Bleeding ──
  bleeding_intensity?: string;
  blood_color?: string;
  bleeding_smell?: string;
  has_clots?: boolean;
  spotting?: boolean;

  // ── Pain & physical symptoms (all use the `pain_intensity` enum) ──
  headache_intensity?: string;
  stomach_ache_intensity?: string;
  pelvic_pain_intensity?: string;
  breast_pain_intensity?: string;
  back_pain_intensity?: string;
  ovarian_pain_intensity?: string;
  nausea_intensity?: string;
  bloating_intensity?: string;
  breast_sensitivity_intensity?: string;
  urination_burning_intensity?: string;

  // ── Digestion & appetite ──
  appetite_change?: string;
  diarrhea?: boolean;
  constipation?: boolean;
  food_craving?: boolean;

  // ── Mood ──
  moods?: string[];

  // ── Sleep ──
  sleep_duration?: string;
  sleep_quality?: string;

  // ── Skin & body ──
  acne?: boolean;
  oily_skin?: boolean;
  hair_loss?: boolean;
  swelling?: boolean;
  fatigue?: boolean;
  dizziness?: boolean;
  hot_flashes?: boolean;
  chills?: boolean;

  // ── Discharge ──
  discharge_texture?: string;
  discharge_amount?: string;
  discharge_smell?: string;
  discharge_itching?: boolean;
  discharge_burning?: boolean;
  frequent_urination?: boolean;

  // ── Intimate / urinary ──
  vaginal_dryness?: boolean;
  vaginal_burning?: boolean;
  vaginal_itching?: boolean;
  vaginal_smell_change?: boolean;
  urination_change?: string;

  // ── Sexual activity ──
  sexual_activities?: string[];

  // ── Measurements ──
  weight?: number;
  basal_body_temperature?: number;

  // ── Free text ──
  notes?: string;
}

/** Every logged field (everything except the date key). */
export type HealthLogField = keyof Omit<HealthLogInput, 'log_date'>;

/**
 * How a single field is captured in the UI. Enum-backed controls point at the
 * `EnumKey` whose options they render; `measure`/`note`/`bool` are self-describing.
 */
export type FieldControl =
  | { kind: 'chips'; enumKey: EnumKey } // single-select from an enum
  | { kind: 'degree'; enumKey: EnumKey } // single-select intensity (segmented)
  | { kind: 'multi'; enumKey: EnumKey } // multi-select from an enum
  | { kind: 'bool' } // on/off toggle
  | { kind: 'measure'; unit: 'kg' | 'celsius'; min: number; max: number; step: number }
  | { kind: 'note' }; // free-text

export interface FieldDef {
  key: HealthLogField;
  control: FieldControl;
}

/**
 * The log categories (Figma "Add Log" step sheets). A closed union so the i18n
 * keys `log.categories.<key>` / `log.categoryHint.<key>` stay compile-checked.
 */
export type CategoryKey =
  | 'bleeding'
  | 'pain'
  | 'digestion'
  | 'mood'
  | 'sleep'
  | 'body'
  | 'discharge'
  | 'intimate'
  | 'sexual'
  | 'measure'
  | 'notes';

/** A logical group of fields, shown as one card and edited in one bottom sheet. */
export interface CategoryDef {
  /** i18n + style lookup key (`log.categories.<key>`). */
  key: CategoryKey;
  fields: FieldDef[];
}
