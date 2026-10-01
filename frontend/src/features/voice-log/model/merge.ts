import type { LogCategory, LogDayValues, LogItemValue, LogParamValue } from '@/entities/health-log';

import type { VoiceSuggestion } from '../api/voice';

/**
 * Pure merge of voice suggestions into the log sheet's draft (`LogDayValues`, the PUT body shape). A
 * suggestion adds to what is already logged — a multi item joins the list, an items entry is set on its
 * own key — and replaces only scalar params (single option, number, bool). No React, no locale.
 */

export interface MergedParam {
  category: string;
  param: string;
  value: LogParamValue;
}

/** `category.param` of a suggestion (the PUT `voice_params` key). */
export function paramKey(s: Pick<VoiceSuggestion, 'category' | 'param'>): string {
  return `${s.category}.${s.param}`;
}

function paramType(categories: readonly LogCategory[], category: string, param: string) {
  return categories.find((c) => c.code === category)?.params.find((p) => p.code === param)?.type ?? null;
}

function itemsOf(value: LogParamValue | undefined): Record<string, LogItemValue> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {};
  const out: Record<string, LogItemValue> = {};
  for (const [k, v] of Object.entries(value)) {
    if (v && typeof v === 'object' && typeof (v as LogItemValue).level === 'string') out[k] = v as LogItemValue;
  }
  return out;
}

/** The new value of one suggestion's param on top of `current`, or null when it can't apply. */
export function mergeValue(type: string | null, s: VoiceSuggestion, current: LogParamValue | undefined): LogParamValue | null {
  switch (type) {
    case 'single':
      return typeof s.value === 'string' ? s.value : null;
    case 'multi': {
      if (!s.item) return null;
      const list = Array.isArray(current) ? current : [];
      return list.includes(s.item) ? list : [...list, s.item];
    }
    case 'items': {
      if (!s.item || typeof s.value !== 'string') return null;
      const items = itemsOf(current);
      const before = items[s.item];
      const score = before && before.level === s.value ? before.score : null;
      return { ...items, [s.item]: { level: s.value, score } };
    }
    case 'number':
    case 'integer':
      return typeof s.value === 'number' && Number.isFinite(s.value) ? s.value : null;
    case 'bool':
      return typeof s.value === 'boolean' ? s.value : null;
    default:
      return null;
  }
}

/**
 * Every suggestion merged in order (several items of one param accumulate). Returns the changed params
 * with their final values; a suggestion whose param the sheet doesn't show (another mode, hidden category)
 * is skipped.
 */
export function mergeSuggestions(
  values: LogDayValues,
  suggestions: readonly VoiceSuggestion[],
  categories: readonly LogCategory[],
): MergedParam[] {
  const next: Record<string, LogParamValue> = {};
  const order: string[] = [];
  for (const s of suggestions) {
    const key = paramKey(s);
    const current = key in next ? next[key] : values[s.category]?.[s.param];
    const value = mergeValue(paramType(categories, s.category, s.param), s, current);
    if (value === null) continue;
    if (!(key in next)) order.push(key);
    next[key] = value;
  }
  return order.map((key) => {
    const [category, param] = key.split('.');
    return { category, param, value: next[key] };
  });
}
