import type { ConfigField } from '../api/clinical-config';

/** Field names the value cell prints as a pair («> 180 / > 120») rather than labelled numbers. */
const PAIRS: Record<string, string> = { 'systolic,diastolic': ' / ', 'min,max': '–', 'target_max,very_high_from': ' · ' };

/** «180 / 120», «24–38», «13», or «before 5 · after 1» for other multi-field values. */
export function formatValue(
  fields: ConfigField[],
  value: Record<string, number>,
  num: (n: number) => string,
): string {
  const names = fields.map((f) => f.name);
  const present = names.filter((n) => value[n] !== undefined);
  if (present.length === 1) return num(value[present[0]!]!);
  const sep = PAIRS[names.join(',')];
  if (sep) return present.map((n) => num(value[n]!)).join(sep);
  return present.map((n) => `${n} ${num(value[n]!)}`).join(' · ');
}

/** Parses the form's text inputs into numbers; null when any field is empty or not a number. */
export function parseValue(fields: ConfigField[], raw: Record<string, string>): Record<string, number> | null {
  const out: Record<string, number> = {};
  for (const f of fields) {
    const s = (raw[f.name] ?? '').trim();
    const n = Number(s);
    if (s === '' || !Number.isFinite(n)) return null;
    out[f.name] = n;
  }
  return out;
}
