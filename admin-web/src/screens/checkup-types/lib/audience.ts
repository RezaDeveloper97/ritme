/**
 * `checkup_types.audiences` (admin-api.md §12, CB-MENO-01b): the life modes a shared checkup is shown to;
 * empty = everyone. The list API has no audience query, so the list screen filters the (≤ 100) rows here.
 */

/** The life modes (`options.audiences`); a fallback when an older API omits the list. */
export const LIFE_MODES = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen'] as const;

/** `all` = no filter, `everyone` = rows without audiences, otherwise a life mode code. */
export type AudienceFilter = 'all' | 'everyone' | (string & {});

/** Rows the filter keeps: a mode keeps the rows that list it (rows for everyone are under `everyone`). */
export function filterByAudience<T extends { audiences: readonly string[] }>(rows: readonly T[], filter: AudienceFilter): T[] {
  if (filter === 'all') return [...rows];
  if (filter === 'everyone') return rows.filter((r) => r.audiences.length === 0);
  return rows.filter((r) => r.audiences.includes(filter));
}

/** The write value: the ticked modes in the options' order, `null` when none (= everyone). */
export function audiencesBody(selected: readonly string[], options: readonly string[]): string[] | null {
  const set = new Set(selected);
  const ordered = options.filter((o) => set.has(o));
  // Keep a stored mode the options no longer list, so a save never drops it silently (the API answers 422).
  for (const s of selected) if (!options.includes(s) && !ordered.includes(s)) ordered.push(s);
  return ordered.length ? ordered : null;
}

/** The first `audiences` / `audiences.N` error of a 422 bag. */
export function audiencesError(errors: Record<string, string[]> | undefined): string | undefined {
  if (!errors) return undefined;
  const key = Object.keys(errors).find((k) => k === 'audiences' || k.startsWith('audiences.'));
  return key ? errors[key]?.[0] : undefined;
}
