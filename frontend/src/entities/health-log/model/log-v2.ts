/**
 * Log taxonomy v2 (B-N3-01/02): the shapes of `GET /logs/taxonomy`, `GET /logs/days*` and
 * `GET /logs/preferences` (backend-go/api/openapi.yaml, internal/healthlog/taxonomy/view.go).
 * The taxonomy is code on the server and labels are translation data, so the client renders it
 * generically from these types — no category or option list is hardcoded here.
 */

/** How a param is entered and stored. `link` is a read-only tile owned by another feature. */
export type LogParamType =
  | 'single'
  | 'multi'
  | 'items'
  | 'number'
  | 'integer'
  | 'text'
  | 'text_items'
  | 'bool'
  | 'link';

export interface LogLabeled {
  value: string;
  label: string;
}

export interface LogOption extends LogLabeled {
  /** Limits the option to these modes (`null` = wherever the param shows). */
  modes: string[] | null;
  /** A value only legacy data carries: labelled, never offered for new input. */
  legacyOnly: boolean;
}

export interface LogRange {
  min: number;
  max: number;
}

export interface LogParam {
  code: string;
  type: LogParamType;
  label: string;
  modes: string[];
  /** Optional detail behind «جزئیات» (colour, clots, odour…). */
  detail: boolean;
  /** Always worth a doctor's look in this mode set (bleeding in pregnancy). */
  alert: boolean;
  /** single/multi options, or the items of an items/text_items param. */
  options: LogOption[];
  /** items: the level codes, labelled. */
  levels: LogLabeled[];
  /** items: optional per-item score (pain 1–10). */
  score: LogRange | null;
  /** number/integer bounds. */
  range: LogRange | null;
  /** number: decimal places. */
  scale: number | null;
  unit: LogLabeled | null;
  maxLength: number | null;
  /** items/text_items: free item codes (custom items, care reminder ids). */
  dynamic: boolean;
  /** link: the feature that owns the data (`kick_counter`, `feeding`, …). */
  source: string | null;
}

export interface LogCategory {
  code: string;
  label: string;
  group: LogLabeled;
  modes: string[];
  /** Per-mode availability notes («بعد از ۶ هفته»). */
  conditions: Record<string, LogLabeled>;
  params: LogParam[];
}

export interface LogTaxonomy {
  /** The mode the list is filtered for (`null` = every mode). */
  mode: string | null;
  categories: LogCategory[];
}

/** One item of an `items` param: a level code and an optional score. */
export interface LogItemValue {
  level: string;
  score: number | null;
}

/**
 * A stored param value, in the PUT body shape: single → code, multi → codes, items → {item: {level, score}},
 * number/integer → number, text → string, text_items → {item: text}, bool → boolean.
 */
export type LogParamValue =
  | string
  | string[]
  | number
  | boolean
  | Record<string, LogItemValue>
  | Record<string, string>;

/** A day's values: `{category: {param: value}}`. */
export type LogDayValues = Record<string, Record<string, LogParamValue>>;

export interface LogDay {
  date: string;
  categories: LogDayValues;
}

export interface LogDaysRange {
  from: string;
  to: string;
  days: LogDay[];
}

export interface LogPrefCategory {
  code: string;
  label: string;
  hidden: boolean;
  pinned: boolean;
  /** The param hosting the user's custom items of this category, if any. */
  customParam: string | null;
}

export interface LogCustomItem {
  id: number;
  /** Item code in health_log_entries (`custom_12`). */
  code: string;
  category: string;
  param: string;
  label: string;
}

export interface LogPreferences {
  mode: string;
  phase: string | null;
  isDefault: boolean;
  maxPinned: number;
  /** Quick tile keys in order: a category code or `category.param`. */
  pinned: string[];
  /** Every category of the mode in the user's order. */
  categories: LogPrefCategory[];
  maxCustomItems: number;
  customItems: LogCustomItem[];
}

/** The PUT body's per-param change: a value, or `null` to clear it. */
export type LogDayChanges = Record<string, Record<string, LogParamValue | null>>;
