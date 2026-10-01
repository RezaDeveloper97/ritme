import type { SymptomHighlight, SymptomsReport } from '@/entities/analysis';
import type { Tone } from '@/shared/ui';

/** Series tones in trend order (An_Symptoms: سردرد violet, گرفتگی pink, …). */
export const SERIES_TONES: readonly Tone[] = ['brand', 'bloom', 'warm', 'data', 'period'];

/** At most this many lanes on the trend chart at once. */
export const MAX_SELECTED = 3;
/** Lanes on by default (the artboard opens with two). */
export const DEFAULT_SELECTED = 2;

export function toneOf(keys: readonly string[], key: string): Tone {
  const i = keys.indexOf(key);
  return SERIES_TONES[(i < 0 ? 0 : i) % SERIES_TONES.length];
}

/** Tone of a pattern row: its trend tone when it is on the trend, else the next free tone by row. */
export function patternTone(trendKeys: readonly string[], key: string, row: number): Tone {
  return trendKeys.includes(key) ? toneOf(trendKeys, key) : SERIES_TONES[(trendKeys.length + row) % SERIES_TONES.length];
}

/**
 * The selected trend keys still present in this report, in trend order; falls
 * back to the first {@link DEFAULT_SELECTED} keys when nothing chosen survives
 * a range change.
 */
export function activeKeys(trendKeys: readonly string[], chosen: readonly string[] | null): string[] {
  const kept = trendKeys.filter((k) => chosen?.includes(k));
  return kept.length ? kept : trendKeys.slice(0, DEFAULT_SELECTED);
}

/** Toggle one chip: keeps at least one lane and at most {@link MAX_SELECTED}. */
export function toggleKey(trendKeys: readonly string[], current: readonly string[], key: string): string[] {
  if (current.includes(key)) return current.length > 1 ? current.filter((k) => k !== key) : [...current];
  const next = [...current, key];
  const trimmed = next.length > MAX_SELECTED ? next.slice(next.length - MAX_SELECTED) : next;
  return trendKeys.filter((k) => trimmed.includes(k));
}

/** Message variant of a pattern window (`analysis.hub.symptoms.highlight.*`, the wording the hub and the top finding share). */
export function highlightKey(h: Pick<SymptomHighlight, 'relation' | 'startDay' | 'endDay'>): 'before_period' | 'early' | 'mid' | 'mid_day' {
  if (h.relation === 'before_period' || h.relation === 'early') return h.relation;
  return h.startDay === h.endDay ? 'mid_day' : 'mid';
}

/** Label of each trend key (from the top list, which carries the localized names). */
export function labelsOf(r: SymptomsReport): Record<string, string> {
  const out: Record<string, string> = {};
  for (const s of r.top) out[s.key] = s.label;
  for (const p of r.pattern.items) out[p.key] ??= p.label;
  return out;
}
