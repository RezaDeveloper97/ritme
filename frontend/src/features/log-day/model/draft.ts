import type {
  LogCategory,
  LogDayChanges,
  LogDayValues,
  LogItemValue,
  LogOption,
  LogParam,
  LogParamValue,
} from '@/entities/health-log';

/**
 * Pure draft logic of the log sheet (B-N3-03): the sheet edits a local copy of the day (`LogDayValues`,
 * the PUT body shape) and saves only what changed in one `PUT /logs/days/{date}`. No React, no locale —
 * labels and number formatting come in through {@link LabelContext}.
 */

export type ItemsValue = Record<string, LogItemValue>;
export type TextItemsValue = Record<string, string>;

/** Whether a value means "nothing logged" (cleared params are dropped from the draft). */
export function isEmptyValue(value: LogParamValue | null | undefined): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === 'string') return value.trim() === '';
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === 'object') return Object.keys(value).length === 0;
  return false;
}

function sortedJson(value: unknown): string {
  if (Array.isArray(value)) return JSON.stringify([...value].map(String).sort());
  if (value && typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) => a.localeCompare(b));
    return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${sortedJson(v)}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

/** Deep equality where a multi list's order and an object's key order don't matter. */
export function valuesEqual(a: LogParamValue | null | undefined, b: LogParamValue | null | undefined): boolean {
  const ea = isEmptyValue(a);
  const eb = isEmptyValue(b);
  if (ea || eb) return ea && eb;
  return sortedJson(a) === sortedJson(b);
}

export function getValue(values: LogDayValues, category: string, param: string): LogParamValue | undefined {
  return values[category]?.[param];
}

/** Sets (or, for an empty value, removes) one param — returns a new object, never mutates. */
export function setValue(
  values: LogDayValues,
  category: string,
  param: string,
  value: LogParamValue | null,
): LogDayValues {
  const params = { ...(values[category] ?? {}) };
  if (isEmptyValue(value)) delete params[param];
  else params[param] = value as LogParamValue;
  const next = { ...values };
  if (Object.keys(params).length) next[category] = params;
  else delete next[category];
  return next;
}

/** The PUT body: every param whose draft differs from the saved day — its new value, or `null` to clear. */
export function diffDay(saved: LogDayValues, draft: LogDayValues): LogDayChanges {
  const changes: LogDayChanges = {};
  const cats = new Set([...Object.keys(saved), ...Object.keys(draft)]);
  for (const cat of cats) {
    const params = new Set([...Object.keys(saved[cat] ?? {}), ...Object.keys(draft[cat] ?? {})]);
    for (const param of params) {
      const before = saved[cat]?.[param];
      const after = draft[cat]?.[param];
      if (valuesEqual(before, after)) continue;
      (changes[cat] ??= {})[param] = isEmptyValue(after) ? null : (after as LogParamValue);
    }
  }
  return changes;
}

export function hasChanges(changes: LogDayChanges): boolean {
  return Object.keys(changes).length > 0;
}

// ── Typed readers ──────────────────────────────────────────────────

export function asString(value: LogParamValue | undefined): string | null {
  return typeof value === 'string' ? value : null;
}

export function asList(value: LogParamValue | undefined): string[] {
  return Array.isArray(value) ? value : [];
}

export function asNumber(value: LogParamValue | undefined): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

export function asBool(value: LogParamValue | undefined): boolean | null {
  return typeof value === 'boolean' ? value : null;
}

export function asItems(value: LogParamValue | undefined): ItemsValue {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  const out: ItemsValue = {};
  for (const [k, v] of Object.entries(value)) {
    if (v && typeof v === 'object' && typeof (v as LogItemValue).level === 'string') out[k] = v as LogItemValue;
  }
  return out;
}

export function asTextItems(value: LogParamValue | undefined): TextItemsValue {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  const out: TextItemsValue = {};
  for (const [k, v] of Object.entries(value)) if (typeof v === 'string') out[k] = v;
  return out;
}

// ── Editing helpers ─────────────────────────────────────────────────

export function toggleInList(list: readonly string[], code: string): string[] {
  return list.includes(code) ? list.filter((c) => c !== code) : [...list, code];
}

/** Pain levels (`mild|moderate|severe`) vs yes/no symptoms: what a freshly picked item is logged as. */
export function defaultLevel(param: LogParam): string {
  const codes = param.levels.map((l) => l.value);
  if (codes.includes('yes')) return 'yes';
  if (codes.includes('moderate')) return 'moderate';
  return codes[0] ?? 'yes';
}

/** Whether an items param carries a graded level (pain) rather than a yes/no. */
export function isGraded(param: LogParam): boolean {
  return param.type === 'items' && param.levels.some((l) => l.value === 'mild') && !param.levels.some((l) => l.value === 'yes');
}

export function toggleItem(items: ItemsValue, code: string, level: string): ItemsValue {
  const next = { ...items };
  if (next[code] && next[code].level !== 'no') delete next[code];
  else next[code] = { level, score: null };
  return next;
}

/** Sets the level of every picked item (the «شدت» row under pain locations); a score that disagrees is dropped. */
export function setAllLevels(items: ItemsValue, level: string): ItemsValue {
  const next: ItemsValue = {};
  for (const [k, v] of Object.entries(items)) {
    const keepScore = v.score !== null && levelFromScore(v.score) === level;
    next[k] = { level, score: keepScore ? v.score : null };
  }
  return next;
}

/** The shared level of the picked items, or `null` when they differ / none is picked. */
export function commonLevel(items: ItemsValue): string | null {
  const levels = new Set(Object.values(items).map((v) => v.level));
  return levels.size === 1 ? [...levels][0] : null;
}

/** 1–10 pain score → level: 1–3 mild, 4–6 moderate, 7–10 severe. */
export function levelFromScore(score: number): 'mild' | 'moderate' | 'severe' {
  if (score <= 3) return 'mild';
  if (score <= 6) return 'moderate';
  return 'severe';
}

export function setItemScore(items: ItemsValue, code: string, score: number): ItemsValue {
  return { ...items, [code]: { level: levelFromScore(score), score } };
}

/** Options a user may pick now (legacy-only values never are; the taxonomy is already mode-filtered). */
export function pickableOptions(param: LogParam, mode: string | null): LogOption[] {
  return param.options.filter((o) => !o.legacyOnly && (o.modes === null || mode === null || o.modes.includes(mode)));
}

/** Parses a typed number in any digit script with `.`, `٫` or `,` as the decimal point; `null` if not a number. */
export function parseDecimal(input: string): number | null {
  const ascii = input
    .trim()
    .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/[٫,]/g, '.');
  if (!/^-?\d+(\.\d+)?$/.test(ascii) && !/^-?\d+\.$/.test(ascii)) return null;
  const n = Number(ascii.replace(/\.$/, ''));
  return Number.isFinite(n) ? n : null;
}

/** Rounds to the param's scale (number) or to an integer, clamped into its range. */
export function normalizeNumber(param: LogParam, n: number): number {
  const scale = param.type === 'integer' ? 0 : (param.scale ?? 2);
  const f = 10 ** scale;
  let v = Math.round(n * f) / f;
  if (param.range) v = Math.min(param.range.max, Math.max(param.range.min, v));
  return v;
}

// ── Labels & summaries ─────────────────────────────────────────────

export interface LabelContext {
  /** Labels of the user's custom items and care-reminder ids, by item code. */
  extraLabels: Record<string, string>;
  /** «ثبت شده» — a stored value the current taxonomy can't name (legacy, another mode). */
  unknown: string;
  /** Formats a number in the locale with `scale` decimals («۵۸٫۴»). */
  formatNumber: (value: number, scale: number) => string;
}

/** One logged thing: `label` for the footer list, `summary` for the accordion line («۵۸٫۴ کیلو»). */
export interface LogEntry {
  key: string;
  label: string;
  summary: string;
}

export function optionLabel(param: LogParam, code: string, ctx: LabelContext): string {
  return param.options.find((o) => o.value === code)?.label ?? ctx.extraLabels[code] ?? ctx.unknown;
}

export function levelLabel(param: LogParam, code: string): string | null {
  return param.levels.find((l) => l.value === code)?.label ?? null;
}

function trimText(text: string, max = 32): string {
  const t = text.trim().replace(/\s+/g, ' ');
  return t.length > max ? `${t.slice(0, max - 1)}…` : t;
}

/** The logged entries of one param value (nothing for an empty value, a `false` bool or a link). */
export function paramEntries(category: string, param: LogParam, value: LogParamValue | undefined, ctx: LabelContext): LogEntry[] {
  if (isEmptyValue(value)) return [];
  const key = (suffix = '') => `${category}.${param.code}${suffix}`;
  switch (param.type) {
    case 'single': {
      const code = asString(value);
      if (!code) return [];
      const label = optionLabel(param, code, ctx);
      return [{ key: key(), label, summary: label }];
    }
    case 'multi':
      return asList(value).map((code) => {
        const label = optionLabel(param, code, ctx);
        return { key: key(`.${code}`), label, summary: label };
      });
    case 'items':
      return Object.entries(asItems(value))
        .filter(([, v]) => v.level !== 'no')
        .map(([code, v]) => {
          const label = optionLabel(param, code, ctx);
          const level = isGraded(param) ? levelLabel(param, v.level) : null;
          return { key: key(`.${code}`), label, summary: level ? `${label} · ${level}` : label };
        });
    case 'number':
    case 'integer': {
      const n = asNumber(value);
      if (n === null) return [];
      const scale = param.type === 'integer' ? 0 : Math.min(param.scale ?? 1, decimals(n));
      const unit = param.unit ? ` ${param.unit.label}` : '';
      return [{ key: key(), label: param.label, summary: `${ctx.formatNumber(n, scale)}${unit}` }];
    }
    case 'text': {
      const text = asString(value);
      return text ? [{ key: key(), label: param.label, summary: trimText(text) }] : [];
    }
    case 'text_items':
      return Object.keys(asTextItems(value)).map((code) => {
        const label = optionLabel(param, code, ctx);
        return { key: key(`.${code}`), label, summary: label };
      });
    case 'bool':
      return asBool(value) ? [{ key: key(), label: param.label, summary: param.label }] : [];
    default:
      return [];
  }
}

function decimals(n: number): number {
  const s = String(n);
  const i = s.indexOf('.');
  return i < 0 ? 0 : s.length - i - 1;
}

export function categoryEntries(category: LogCategory, values: LogDayValues, ctx: LabelContext): LogEntry[] {
  return category.params.flatMap((p) => paramEntries(category.code, p, values[category.code]?.[p.code], ctx));
}

/** Everything logged on the day, in category order. */
export function dayEntries(categories: readonly LogCategory[], values: LogDayValues, ctx: LabelContext): LogEntry[] {
  return categories.flatMap((c) => categoryEntries(c, values, ctx));
}

// ── Tiles & search ──────────────────────────────────────────────────

/** A quick tile key: a category code («pain») or `category.param` («measurements.weight»). */
export function parseTileKey(key: string): { category: string; param: string | null } {
  const i = key.indexOf('.');
  return i < 0 ? { category: key, param: null } : { category: key.slice(0, i), param: key.slice(i + 1) };
}

/**
 * Folds what people type so «سردرد» finds «سردرد» whatever the keyboard: case, Arabic ي/ك vs Persian
 * ی/ک, ZWNJ and diacritics.
 */
export function normalizeSearch(text: string): string {
  return text
    .toLowerCase()
    .replace(/[يى]/g, 'ی')
    .replace(/ك/g, 'ک')
    .replace(/[‌‏‎ً-ْ]/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

/** Whether a category matches the search: its label, a param label or an option label. */
export function matchesSearch(category: LogCategory, query: string, extraLabels: readonly string[] = []): boolean {
  const q = normalizeSearch(query);
  if (!q) return true;
  const haystack = [
    category.label,
    ...category.params.flatMap((p) => [p.label, ...p.options.map((o) => o.label)]),
    ...extraLabels,
  ];
  return haystack.some((s) => normalizeSearch(s).includes(q));
}
