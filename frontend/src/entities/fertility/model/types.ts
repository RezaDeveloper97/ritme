/**
 * Fertility tracking for TTC users (M5): quick tiles, the day log, the BBT chart
 * and the fertile-window insights. Contract: docs/fertility-ttc/README.md
 * (`/api/v1/fertility/*`, Go only).
 *
 * Personal health data (§11) — display only, never logged, never put in
 * analytics, error reports or URLs beyond what the API needs.
 *
 * Dates cross the boundary as `Y-m-d` and are converted to the locale calendar
 * only for display (`shared/lib/date`, §7). Server `label`s are already
 * localized (Accept-Language); the enum `value`s drive selection state.
 *
 * Every enum below is nullable in the parsed shape: `null` means "not logged"
 * (the «ثبت نشه» chip) — and also what an enum value this bundle doesn't know
 * yet parses to, so a newer backend can never crash an older bundle.
 */

/** LH ovulation test strip (`fertility_logs.lh_test`). */
export const LH_RESULTS = ['negative', 'faint', 'positive'] as const;
export type LhResult = (typeof LH_RESULTS)[number];

/** Cervical mucus (`fertility_logs.cervical_mucus`), driest → most fertile. */
export const CERVICAL_MUCUS = ['dry', 'sticky', 'creamy', 'egg_white'] as const;
export type CervicalMucus = (typeof CERVICAL_MUCUS)[number];

/** `daily_health_logs.intercourse_type`; null = «ثبت نشه». */
export const INTERCOURSE_TYPES = ['unprotected', 'protected'] as const;
export type IntercourseType = (typeof INTERCOURSE_TYPES)[number];

/**
 * Symptom chips, each backed by an existing `daily_health_logs` column:
 * ovarian_pain → `ovarian_pain_intensity`, bloating → `bloating_intensity`,
 * breast_sensitivity → `breast_sensitivity_intensity`, spotting → `spotting`.
 */
export const FERTILITY_SYMPTOMS = ['ovarian_pain', 'bloating', 'breast_sensitivity', 'spotting'] as const;
export type FertilitySymptom = (typeof FERTILITY_SYMPTOMS)[number];

/** The cycle view's `fertility_level` (no second prediction model). */
export const CHANCE_LEVELS = ['none', 'low', 'medium', 'high', 'peak'] as const;
export type ChanceLevel = (typeof CHANCE_LEVELS)[number];

/** Chance card bar count, 0–5. */
export const CHANCE_MAX_BARS = 5;

export const BBT_PHASES = ['pre_shift', 'post_shift'] as const;
export type BbtPhase = (typeof BBT_PHASES)[number];

/** `GET /fertility/bbt?range=` — how many cycles the chart spans. */
export const BBT_RANGES = [1, 3, 6] as const;
export type BbtRange = (typeof BBT_RANGES)[number];

export const INSIGHT_CONFIDENCE = ['low', 'medium', 'high'] as const;
export type InsightConfidence = (typeof INSIGHT_CONFIDENCE)[number];

/** Evidence row strength: قوی / متوسط / ندارد. */
export const EVIDENCE_STRENGTHS = ['strong', 'medium', 'none'] as const;
export type EvidenceStrength = (typeof EVIDENCE_STRENGTHS)[number];

/** A server-localized enum value (`{value, label}`). */
export interface LabeledValue<T extends string> {
  value: T | null;
  /** Localized label; null when the server sent none (render from i18n). */
  label: string | null;
}

export interface FertilityChance {
  level: ChanceLevel | null;
  label: string | null;
  /** 0–5 filled bars. */
  bars: number;
}

/** `GET /fertility/today` — the three home tiles + the chance level. */
export interface FertilityToday {
  date: string;
  cycleDay: number | null;
  chance: FertilityChance;
  lh: LabeledValue<LhResult>;
  /** °C, 2 decimals; null = not logged today. */
  bbt: number | null;
  intercourse: LabeledValue<IntercourseType>;
}

/** `GET|PUT /fertility/days/{date}` — the merged day (fertility_logs + daily_health_logs). */
export interface FertilityDay {
  date: string;
  cycleDay: number | null;
  lh: LhResult | null;
  mucus: CervicalMucus | null;
  /** °C, 2 decimals. */
  bbt: number | null;
  /** `HH:MM` the temperature was taken; null = not recorded. */
  bbtTime: string | null;
  intercourse: IntercourseType | null;
  /** Known symptoms only, de-duplicated, in {@link FERTILITY_SYMPTOMS} order. */
  symptoms: FertilitySymptom[];
  note: string | null;
  chance: FertilityChance | null;
}

/**
 * `PUT /fertility/days/{date}` input. Every field is optional: omitted =
 * leave as is, explicit `null` = clear (for `symptoms`, `[]` clears them all).
 */
export interface FertilityDayInput {
  lh?: LhResult | null;
  mucus?: CervicalMucus | null;
  bbt?: number | null;
  bbtTime?: string | null;
  intercourse?: IntercourseType | null;
  symptoms?: FertilitySymptom[];
  note?: string | null;
}

export interface BbtPoint {
  cycleDay: number;
  date: string;
  /** °C. */
  value: number;
}

export interface BbtCycle {
  /** First day of the cycle (`Y-m-d`); null when the server omits it. */
  startDate: string | null;
  points: BbtPoint[];
  /** Max of the 6 readings before the shift (3-over-6); null until resolvable. */
  coverline: number | null;
  /** Cycle-day span of the fertile window (inclusive). */
  fertileWindow: { fromDay: number; toDay: number } | null;
  /** Cycle day of the first high reading of a confirmed shift. */
  shiftDay: number | null;
  phase: BbtPhase | null;
}

export interface BbtStats {
  /** Average of the pre-ovulation readings, °C. */
  preOvulationAvg: number | null;
  loggedDays: number;
  cycleDaysSoFar: number;
  gaps: number;
}

export interface FertilityTip {
  title: string | null;
  body: string;
}

/** `GET /fertility/bbt?range=1|3|6`. The current cycle is `cycles[0]`. */
export interface FertilityBbt {
  range: BbtRange;
  cycles: BbtCycle[];
  stats: BbtStats;
  /** Shift days of the previous cycles, most recent first. */
  pastShiftDays: number[];
  tip: FertilityTip | null;
}

export interface InsightEvidence {
  /** Server key (e.g. `cycles`, `bbt_shift`, `lh`) — picks the row icon. */
  key: string;
  title: string;
  detail: string | null;
  strength: EvidenceStrength | null;
}

export interface OvulationHistoryRow {
  monthLabel: string;
  ovulationDay: number | null;
  /** First day of that cycle (`Y-m-d`); null when the server omits it. */
  cycleStart: string | null;
}

/** `GET /fertility/insights`. */
export interface FertilityInsights {
  cyclesUsed: number;
  /** Estimated fertile window (`Y-m-d`); null when there is no anchor yet. */
  window: { start: string; end: string; ovulation: string | null } | null;
  confidence: InsightConfidence | null;
  evidence: InsightEvidence[];
  history: OvulationHistoryRow[];
  tips: string[];
}
