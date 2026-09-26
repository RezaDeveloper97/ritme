/** Drop empty translations; `null` when nothing is left (the API clears the column). */
export function cleanTranslations(value: Record<string, string>): Record<string, string> | null {
  const out: Record<string, string> = {};
  for (const [code, text] of Object.entries(value)) if (text.trim()) out[code] = text.trim();
  return Object.keys(out).length ? out : null;
}

/** Move `from` to `to` (a new array; out-of-range moves are no-ops). */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  const next = [...items];
  if (from < 0 || from >= next.length || to < 0 || to >= next.length) return next;
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item as T);
  return next;
}

/**
 * The key of a new week task: `w<week>_<n>` with the first free n. Keys are stored in
 * users' `done_task_keys`, so an existing task keeps its key even when reordered.
 */
export function nextTaskKey(week: number, taken: readonly string[]): string {
  for (let n = 1; ; n++) {
    const key = `w${week}_${n}`;
    if (!taken.includes(key)) return key;
  }
}

/** Blank → null for the ASCII measurement inputs. */
export const blankOrNull = (v: string): string | null => (v.trim() === '' ? null : v.trim());
