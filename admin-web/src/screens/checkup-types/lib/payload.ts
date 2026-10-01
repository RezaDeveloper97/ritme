import type { CheckupType } from '../api/checkup-types';

/** Move `from` to `to` (a new array). */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  const next = [...items];
  if (from < 0 || from >= next.length || to < 0 || to >= next.length) return next;
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item as T);
  return next;
}

/** Drop empty translations; `null` when nothing is left (the API clears the column). */
export function cleanTranslations(value: Record<string, string>): Record<string, string> | null {
  const out: Record<string, string> = {};
  for (const [code, text] of Object.entries(value)) if (text.trim()) out[code] = text.trim();
  return Object.keys(out).length ? out : null;
}

/**
 * A full PUT body from a stored row. Booleans are `$request->boolean()` on the API
 * (absent = false), so every save — the list's active toggle included — sends both.
 */
export function rowToBody(row: CheckupType): Record<string, unknown> {
  return {
    title: row.title,
    subtitle: cleanTranslations(row.subtitle),
    why: cleanTranslations(row.why),
    category: row.category,
    performed_by: row.performed_by,
    icon: row.icon,
    tone: row.tone,
    interval_months: row.interval_months,
    interval_months_max: row.interval_months_max,
    age_min: row.age_min,
    age_max: row.age_max,
    cycle_day_from: row.cycle_day_from,
    cycle_day_to: row.cycle_day_to,
    remind_lead_days: row.remind_lead_days,
    prep_steps: row.prep_steps,
    guide_steps: row.guide_steps,
    finding_options: row.finding_options.map((f) => ({ key: f.key, label: f.label, ...(f.exclusive ? { exclusive: true } : {}) })),
    hide_in_pregnancy: row.hide_in_pregnancy,
    audiences: row.audiences.length ? row.audiences : null,
    is_active: row.is_active,
    sort_order: row.sort_order,
    source_note: row.source_note,
  };
}
